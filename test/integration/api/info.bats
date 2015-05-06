#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker info" {
	start_docker 1 --label foo=bar
	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	echo $output
	[[ "${output}" == *"Nodes: 1"* ]]
	[[ "${output}" == *"└ Labels:"*"foo=bar"* ]]
}
