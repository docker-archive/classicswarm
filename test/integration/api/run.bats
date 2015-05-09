#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker run" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -d --name test_container busybox sleep 100

	# verify, container is running
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]"
}
