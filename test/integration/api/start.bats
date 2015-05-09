#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker start" {
	start_docker_with_busybox 2
	swarm_manage
	# create
	docker_swarm create --name test_container busybox sleep 1000

	# make sure created container exists
	# new created container has no status
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# start
	docker_swarm start test_container

	# Verify
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]"
}
