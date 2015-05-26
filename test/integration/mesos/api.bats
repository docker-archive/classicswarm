#!/usr/bin/env bats

load mesos_helpers

function teardown() {
	swarm_manage_cleanup
}

@test "docker info" {
	swarm_manage_mesos
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *'Offers'* ]]
}


