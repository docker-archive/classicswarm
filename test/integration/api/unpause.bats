#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker unpause" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name test_container busybox sleep 1000

	# make sure container is up
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"
	[ $(docker_swarm inspect -f '{{ .State.Paused }}' test_container) == 'false' ]

	# pause
	docker_swarm pause test_container

	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Paused }}' test_container) == 'true ']"

	# unpause
	docker_swarm unpause test_container

	# verify
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Paused }}' test_container) == 'false ']"
}
