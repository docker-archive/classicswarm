#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker kill" {
	start_docker_with_busybox 2
	swarm_manage
	# run 
	docker_swarm run -d --name test_container busybox sleep 1000

	# make sure container is up before killing
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"

	# kill
	docker_swarm kill test_container

	# verify
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'false' ]"
	[ $(docker_swarm inspect -f '{{ .State.ExitCode }}' test_container) -eq 137 ] 
}
