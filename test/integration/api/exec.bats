#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker exec" {
	start_docker_with_busybox 2
	swarm_manage
	docker_swarm create --name test_container busybox sleep 100

	# if container is not running, exec will failed
	run docker_swarm exec test_container ls
	[ "$status" -ne 0 ]
	[[ "$output" == *"is not running"* ]]

	docker_swarm start test_container

	# make sure container is up and not paused
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	run docker_swarm exec test_container echo foobar
	[ "$status" -eq 0 ]
	[ "$output" == "foobar" ]
}
