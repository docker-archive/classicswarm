#!/usr/bin/env bats

load helpers

# Address on which the store will listen
STORE_HOST_1=127.0.0.1:8500
STORE_HOST_2=127.0.0.1:8501
STORE_HOST_3=127.0.0.1:8502

# Container name for integration test
CONTAINER_NAME=swarm_leader

# Names for store cluster nodes
NODE_1="node1"
NODE_2="node2"
NODE_3="node3"

# Urls of store cluster nodes
NODE_1_URL="consul://${STORE_HOST_1}/test"
NODE_2_URL="consul://${STORE_HOST_2}/test"
NODE_3_URL="consul://${STORE_HOST_3}/test"

function start_store_cluster() {
	docker_host run -v $(pwd)/discovery/consul/config:/config --name=$NODE_1 -h $NODE_1 -p $STORE_HOST_1:8500 -d progrium/consul -server -bootstrap-expect 3 -config-file=/config/consul.json

	# Grab node_1 address required for other nodes to join the cluster
	JOIN_ENDPOINT=$(docker_host inspect -f '{{.NetworkSettings.IPAddress}}' $NODE_1)

	docker_host run -v $(pwd)/discovery/consul/config:/config --name=$NODE_2 -h $NODE_2 -p $STORE_HOST_2:8500 -d progrium/consul -server -join $JOIN_ENDPOINT -config-file=/config/consul.json

	docker_host run -v $(pwd)/discovery/consul/config:/config --name=$NODE_3 -h $NODE_3 -p $STORE_HOST_3:8500 -d progrium/consul -server -join $JOIN_ENDPOINT -config-file=/config/consul.json

	# Wait for the cluster to be available.
	sleep 2
}

function restart_leader() {
	# TODO find out who is the leader
	docker_host restart -t 5 $NODE_1
}

function stop_store() {
	docker_host rm -f -v $CONTAINER_NAME
}

function stop_store_cluster() {
	docker_host rm -f -v $NODE_1 $NODE_2 $NODE_3
}

function setup() {
	start_store_cluster
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	stop_store_cluster
}

@test "replication options" {
	# Bring up one manager
	# --advertise
	run swarm manage --replication --replication-ttl "4s" --advertise "" "$NODE_1_URL"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--advertise address must be provided when using --leader-election"* ]]

	# --advertise
	run swarm manage --replication --replication-ttl "4s" --advertise 127.0.0.1ab:1bcde "$NODE_1_URL"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--advertise should be of the form ip:port or hostname:port"* ]]

	# --replication-ttl
	run swarm manage --replication --replication-ttl "-20s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$NODE_1_URL"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--replication-ttl should be a positive number"* ]]
}

@test "leader election" {
	local i=${#SWARM_MANAGE_PID[@]}
	local port=$(($SWARM_BASE_PORT + $i))
	local host=127.0.0.1:$port

	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Kill the leader and ensure the replica takes over.
	kill "${SWARM_MANAGE_PID[0]}"
	retry 20 1 eval "docker -H ${SWARM_HOSTS[1]} info | grep -q 'Role: primary'"

	# Add a new replica and make sure it sees the new leader as primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 2)) "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[2]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[1]}"* ]]
}

function containerRunning() {
	local container="$1"
	local node="$2"
	run docker_swarm inspect "$container"
	[ "$status" -eq 0 ]
	[[ "${output}" == *"\"Name\": \"$node\""* ]]
	[[ "${output}" == *"\"Status\": \"running\""* ]]
}

@test "leader election - rescheduling" {
	local i=${#SWARM_MANAGE_PID[@]}
	local port=$(($SWARM_BASE_PORT + $i))
	local host=127.0.0.1:$port

	start_docker_with_busybox 2
	swarm_join "$NODE_1_URL"

	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# c1 on node-0 with reschedule=on-node-failure
	run docker_swarm run -dit --name c1 -e constraint:node==~node-0 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
	[ "$status" -eq 0 ]
	# c2 on node-0 with reschedule=off
	run docker_swarm run -dit --name c2 -e constraint:node==~node-0 --label 'com.docker.swarm.reschedule-policies=["off"]' busybox sh
	[ "$status" -eq 0 ]
	# c3 on node-1
	run docker_swarm run -dit --name c3 -e constraint:node==~node-1 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
	[ "$status" -eq 0 ]

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  3 ]

	# Make sure containers are running where they should.
	containerRunning "c1" "node-0"
	containerRunning "c2" "node-0"
	containerRunning "c3" "node-1"

	# Get c1 swarm id
	swarm_id=$(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' c1)

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	# Wait for the container to be rescheduled
	# c1 should have been rescheduled from node-0 to node-1
	retry 15 1 containerRunning "c1" "node-1"

	# Check swarm id didn't change for c1
	[[ "$swarm_id" == $(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' c1) ]]

	run docker_swarm inspect "$swarm_id"
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# c2 should still be on node-0 since the rescheduling policy was off.
	run docker_swarm inspect c2
	[ "$status" -eq 1 ]

	# c3 should still be on node-1 since it wasn't affected
	containerRunning "c3" "node-1"

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]
}

@test "leader election - store failure" {
	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Fire up a third manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 2)) "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[2]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Stop and start the store holding the leader metadata
	stop_store_cluster
	sleep 3
	start_store_cluster

	# Wait a little bit for the re-election to occur
	# This is specific to Consul (liveness over safety)
	sleep 10

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

@test "leader election - dispatched discovery urls - leader failure" {
	# Bring up one manager, make sure it becomes primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$SWARM_BASE_PORT "$NODE_1_URL"
	run docker -H ${SWARM_HOSTS[0]} info
	[[ "${output}" == *"Role: primary"* ]]

	# Fire up a second manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 1)) "$NODE_2_URL"
	run docker -H ${SWARM_HOSTS[1]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Fire up a third manager. Ensure it's a replica forwarding to the right primary.
	swarm_manage --replication --replication-ttl "4s" --advertise 127.0.0.1:$(($SWARM_BASE_PORT + 2)) "$NODE_3_URL"
	run docker -H ${SWARM_HOSTS[2]} info
	[[ "${output}" == *"Role: replica"* ]]
	[[ "${output}" == *"Primary: ${SWARM_HOSTS[0]}"* ]]

	# Stop and start the store leader
	restart_leader

	# Wait a little bit for the re-election to occur
	# This is specific to Consul (liveness over safety)
	sleep 15

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
