#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
}

@test "docker info" {
	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *'Offers'* ]]
}


