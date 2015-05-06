#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker exec" {
	start_docker_with_busybox 2
	swarm_manage
	run docker_swarm create --name test_container busybox sleep 100
	[ "$status" -eq 0 ]

	# if container is not runing, exec will failed
	run docker_swarm exec test_container ls
	[ "$status" -ne 0 ]
	[[ "$output" == *"is not running"* ]]

	run docker_swarm start test_container
	[ "$status" -eq 0 ]

	# make sure container is up and not paused
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]
	[[ "${lines[1]}" != *"Paused"* ]]	

	run docker_swarm exec test_container ls
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${lines[0]}" == *"bin"* ]]
	[[ "${lines[1]}" == *"dev"* ]]
}
