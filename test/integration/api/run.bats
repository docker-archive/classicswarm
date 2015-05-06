#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker run" {
	start_docker 3
	swarm_manage

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	run docker_swarm run -d --name test_container busybox sleep 100
	[ "$status" -eq 0 ]

	# verify, container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${output}" == *"test_container"* ]]
	[[ "${output}" == *"Up"* ]]
}
