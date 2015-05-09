#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker rm" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm create --name test_container busybox
	
	# make sure container exsists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	docker_swarm rm test_container
	
	# verify
	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq 0 ]
}

@test "docker rm -f" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name test_container busybox sleep 500

	# make sure container exsists and is up
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"

	# rm, remove a running container, return error
	run docker_swarm rm test_container
	[ "$status" -ne 0 ]

	# rm -f, remove a running container
	docker_swarm rm -f test_container

	# verify
	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq 0 ]
}
