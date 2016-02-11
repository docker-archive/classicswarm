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

	eval "docker_swarm info | grep -q -i 'Status: Healthy'"

	# Stop the node and let it fail
	docker_host stop ${DOCKER_CONTAINERS[0]}
	# Wait for swarm to detect node failure
	retry 5 1 eval "docker_swarm info | grep -q -i 'Status: Unhealthy'"

	# Restart node
	docker_host start ${DOCKER_CONTAINERS[0]}
	# Wait for swarm to detect node recovery
	retry 15 1 eval "docker_swarm info | grep -q -i 'Status: Healthy'"
}

@test "node pending and recovery" {
	# Start 1 engine and register it in the file.
	start_docker 1
	setup_discovery_file
	# Stop the node
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Start swarm with the stopped node
	swarm_manage_no_wait "$DISCOVERY"
	retry 2 1 eval "docker_swarm info | grep -q -i 'Status: Pending'"

	# Restart the node
	docker_host start ${DOCKER_CONTAINERS[0]}
	# Wait for swarm to detect node recovery
	retry 15 3 eval "docker_swarm info | grep -q -i 'Status: Healthy'"
}

