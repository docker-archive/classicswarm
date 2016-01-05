#!/usr/bin/env bats

load ../helpers

DISCOVERY_FILE=""
DISCOVERY=""

function setup() {
	# create a blank temp file for discovery
	DISCOVERY_FILE=$(mktemp)
	DISCOVERY="file://$DISCOVERY_FILE"
}

function teardown() {
	swarm_manage_cleanup
	stop_docker
	rm -f "$DISCOVERY_FILE"
}

function setup_discovery_file() {
	rm -f "$DISCOVERY_FILE"
	for host in ${HOSTS[@]}; do
		echo "$host" >> $DISCOVERY_FILE
	done
}

@test "node failure and recovery" {
	# Start 1 engine and register it in the file.
	start_docker 1
	setup_discovery_file
	# Start swarm and check it can reach the node
	swarm_manage --engine-refresh-min-interval "1s" --engine-refresh-max-interval "1s" --engine-failure-retry 2 "$DISCOVERY"

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Status: Healthy"* ]]

	# Stop the node and let it fail
	docker_host stop ${DOCKER_CONTAINERS[0]}
	sleep 4

	# Verify swarm detects node failure
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Status: Unhealthy"* ]]

	# Verify swarm detects recovery
	docker_host start ${DOCKER_CONTAINERS[0]}
	sleep 4
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Status: Healthy"* ]]
}

@test "node pending and recovery" {
	# Start 1 engine and register it in the file.
	start_docker 1
	setup_discovery_file
	# Stop the node
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Start swarm with the stopped node
	swarm_manage_no_wait "$DISCOVERY"
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Status: Pending"* ]]

	# Restart the node and wait for revalidation
	docker_host start ${DOCKER_CONTAINERS[0]}
	sleep 40

	# Verify swarm detects recovery
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Status: Healthy"* ]]
}

