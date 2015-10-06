#!/usr/bin/env bats

load helpers

# Address on which the store will listen
STORE_HOST=127.0.0.1:8500

# Discovery parameter for Swarm
DISCOVERY="consul://${STORE_HOST}/test"

# Container name for integration test
CONTAINER_NAME=swarm_leader

function start_store() {
	docker_host run -v $(pwd)/discovery/consul/config:/config --name=$CONTAINER_NAME -h $CONTAINER_NAME -p $STORE_HOST:8500 -d progrium/consul -server -bootstrap-expect 1 -config-file=/config/consul.json
	# Wait a few seconds for the store to come up.
	sleep 3
}

function stop_store() {
	docker_host rm -f -v $CONTAINER_NAME
}

function setup() {
	start_store
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	stop_store
}

@test "leader election" {
	local i=${#SWARM_MANAGE_PID[@]}
	local port=$(($SWARM_BASE_PORT + $i))
	local host=127.0.0.1:$port

	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Kill the leader and ensure the replica takes over.
	kill "${SWARM_MANAGE_PID[0]}"
	retry 20 1 eval "docker -H ${SWARM_HOSTS[1]} info | grep -q 'Role: primary'"

	# Add a new replica and make sure it sees the new leader as primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 2)) "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[2]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[1]}"* ]]
}

@test "leader election - store failure" {
	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Fire up a third manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 2)) "$DISCOVERY"
	run docker -H ${SWARM_HOSTS[2]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Stop and start the store holding the leader metadata
	stop_store
	sleep 3
	start_store

	# Wait a little bit for the re-election to occur
	# This is specific to Consul (liveness over safety)
	sleep 6

	# Make sure the managers are either in the 'primary' or the 'replica' state.
	for host in "${SWARM_HOSTS[@]}"; do
		retry 120 1 eval "docker -H ${host} info | grep -Eq 'Role: primary|Role: replica'"
	done

	# Find out which one of the node is the Primary and
	# the ones that are Replicas after the store failure
	primary=${SWARM_HOSTS[0]}
	declare -a replicas
	i=0
	for host in "${SWARM_HOSTS[@]}"; do
		run docker -H $host info
		if [[ "${output}" == *"Role: primary"* ]]; then
			primary=$host
		else
			replicas[$((i=i+1))]=$host
		fi
	done

	# Check if we have indeed 2 replicas
	[[ "${#replicas[@]}" -eq 2 ]]

	# Check if the replicas are pointing to the right Primary
	for host in "${replicas[@]}"; do
		run docker -H $host info
		[[ "${output}" == *"Primary: ${primary}"* ]]
	done
}
