#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker port" {
	start_docker_with_busybox 2
	swarm_manage
	docker_swarm run -d -p 8000 --name test_container busybox sleep 500

	# make sure container is up
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# port verify
	run docker_swarm port test_container
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"8000"* ]]
}
