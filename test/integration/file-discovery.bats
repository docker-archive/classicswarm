#!/usr/bin/env bats

load helpers

# create a blank temp file for discovery
DISCOVERY_FILE=$(mktemp)

function teardown() {
	swarm_manage_cleanup
	rm -f $DISCOVERY_FILE
	stop_docker
}

function setup_file_discovery() {
	for host in ${HOSTS[@]}; do
		echo "$host" >> $DISCOVERY_FILE
	done
}

@test "file discovery" {
	start_docker 2
	setup_file_discovery
	swarm_manage file://$DISCOVERY_FILE

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "$output" == *"Nodes: 2 "* ]]
}
