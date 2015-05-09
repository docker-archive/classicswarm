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

	# if container is not runing, exec will failed
	run docker_swarm exec test_container ls
	[ "$status" -ne 0 ]
	[[ "$output" == *"is not running"* ]]

	docker_swarm start test_container

	# make sure container is up and not paused
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"
	[ $(docker_swarm inspect -f '{{ .State.Paused }}' test_container) == 'false' ]

	run docker_swarm exec test_container echo foobar
	[ "$status" -eq 0 ]
	[ "$output" == "foobar" ]
}
