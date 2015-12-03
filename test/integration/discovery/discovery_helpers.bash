#!/bin/bash

load ../helpers

# Returns true if swarm info outputs is empty (0 nodes).
function discovery_check_swarm_info_empty() {
	docker_swarm info | grep -q "Nodes: 0"
}

# Returns true if all nodes have joined the discovery.
function discovery_check_swarm_list() {
	local joined=`swarm list "$1" | wc -l`
	local total="$2"
	[ -z "$total" ] && total="${#HOSTS[@]}"

	echo "${joined} out of ${total} hosts joined discovery"
	[ "$joined" -eq "$total" ]
}
