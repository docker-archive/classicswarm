#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker rename" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	run docker_swarm run -d --name another_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exist
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"test_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"rename_container"* ]]

	# rename container, conflict and fail
	run docker_swarm rename test_container another_container
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Error when allocating new name: Conflict."* ]]

	# rename container, sucessful
	run docker_swarm rename test_container rename_container
	[ "$status" -eq 0 ]

	# verify after, rename 
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"rename_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"test_container"* ]]
}
