#!/bin/bash

load ../helpers

# Returns true if all nodes have joined the swarm.
function discovery_check_swarm_info() {
	docker_swarm info | grep -q "Nodes: ${#HOSTS[@]}"
}

# Returns true if all nodes have joined the discovery.
function discovery_check_swarm_list() {
	local joined=`swarm list "$1" | wc -l`
	local total=${#HOSTS[@]}

	echo "${joined} out of ${total} hosts joined discovery"
	[ "$joined" -eq "$total" ]
}
