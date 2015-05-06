#!/usr/bin/env bats

load helpers

# create a blank temp file for discovery
DISCOVERY_FILE=$(mktemp)

function teardown() {
	swarm_manage_cleanup
	stop_docker
	rm -f "$DISCOVERY_FILE"
}

function setup_file_discovery() {
	rm -f "$DISCOVERY_FILE"
	for host in ${HOSTS[@]}; do
		echo "$host" >> $DISCOVERY_FILE
	done
}

@test "file discovery" {
	# Start 2 engines, register them in a file, then start swarm and make sure
	# it sees them.
	start_docker 2
	setup_file_discovery
	swarm_manage "file://$DISCOVERY_FILE"
	all_nodes_registered_in_swarm

	# Add another engine to the cluster, update the discovery file and make
	# sure it's picked up by swarm.
	start_docker 1
	setup_file_discovery
	retry 10 1 all_nodes_registered_in_swarm
}
