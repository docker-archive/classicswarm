#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker volume" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no volume exist
	run docker_swarm volume
	[ "${#lines[@]}" -eq 1 ]

	# run
	docker_swarm run -d -v=/tmp busybox true

	run docker_swarm volume
	[ "${#lines[@]}" -eq 2 ]

	docker_swarm run -d -v=/tmp busybox true

	run docker_swarm volume
	[ "${#lines[@]}" -eq 3 ]
}

@test "docker volume inspect" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d -v=/tmp busybox true

	run docker_swarm volume ls -q
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm volume inspect ${output}
	[ "${#lines[@]}" -eq 7 ]
	[[ "${output}" == *"\"Driver\": \"local\""* ]]
}