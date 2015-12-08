#!/usr/bin/env bats

load discovery_helpers

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

@test "file discovery: recover engines" {
	# The goal of this test is to ensure swarm can see engines that joined
	# while the manager was stopped.

	# Start 2 engines and register them in the file.
	start_docker 2
	setup_discovery_file
	discovery_check_swarm_list "$DISCOVERY"

	# Then, start a manager and ensure it sees all the engines.
	swarm_manage "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}

@test "file discovery: watch for changes" {
	# The goal of this test is to ensure swarm can see new nodes as they join
	# the cluster.

	# Start a manager with no engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add engines to the cluster and make sure it's picked up by swarm.
	start_docker 2
	setup_discovery_file
	discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}

@test "file discovery: node removal" {
	# The goal of this test is to ensure swarm can handle node removal.

	# Start 2 engines and register them in the file.
	start_docker 2
	setup_discovery_file
	discovery_check_swarm_list "$DISCOVERY"

	# Then, start a manager and ensure it sees all the engines.
	swarm_manage "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info

	# Update the file with only one engine and see if swarm picks it up.
	echo ${HOSTS[0]} > $DISCOVERY_FILE
	discovery_check_swarm_list "$DISCOVERY" 1
	retry 5 1 discovery_check_swarm_info 1
}

@test "file discovery: failure" {
	# The goal of this test is to simulate a failure (file not available) and ensure discovery
	# is resilient to it.

	# Wipe out the discovery file.
	rm -f "$DISCOVERY_FILE"
	
	# Start 2 engines.
	start_docker 2

	# Start a manager. It should keep retrying
	swarm_manage_no_wait "$DISCOVERY"

	# Now create the discovery file.
	setup_discovery_file

	# After a while, `join` and `manage` should see the file.
	discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}
