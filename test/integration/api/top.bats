#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker top" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name test_container busybox sleep 500

	# make sure container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	run docker_swarm top test_container
	[ "$status" -eq 0 ]
	[[ "${lines[0]}" == *"UID"* ]]
	[[ "${lines[0]}" == *"CMD"* ]]
	[[ "${lines[1]}" == *"sleep 500"* ]]
}
