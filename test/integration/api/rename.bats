#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker rename" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name test_container busybox sleep 500

	docker_swarm run -d --name another_container busybox sleep 500

	# make sure container exist
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"test_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"rename_container"* ]]

	# rename container, conflict and fail
	run docker_swarm rename test_container another_container
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Conflict,"* ]]

	# rename container, successful
	docker_swarm rename test_container rename_container

	# verify after, rename 
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"rename_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"test_container"* ]]
}

@test "docker rename conflict" {
	start_docker_with_busybox 1
	swarm_manage

	id=$(docker_swarm create busybox)
	prefix=$(printf "$id" | cut -c 1-4)
	docker_swarm create --name test busybox
	docker_swarm rename test "$prefix"
	run docker_swarm rename test "$id"
	[ "$status" -eq 1 ]
}