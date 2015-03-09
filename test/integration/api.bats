#!/usr/bin/env bats

load helpers

function teardown() {
	stop_manager
	stop_docker
}

@test "docker info should return the number of nodes" {
	start_docker 3
	start_manager
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${lines[1]}" == *"Nodes: 3" ]]
}
