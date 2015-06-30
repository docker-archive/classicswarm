#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker version" {
	# FIXME: No reason here to start docker.
	start_docker 1
	swarm_manage

	# version
	run docker_swarm version
	[ "$status" -eq 0 ]

	# verify
	[[ ${output} =~ 'swarm/' ]]
}
