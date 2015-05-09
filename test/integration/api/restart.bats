#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker restart" {
	start_docker_with_busybox 2
	swarm_manage
	# run 
	docker_swarm run -d --name test_container busybox sleep 1000

	# make sure container is up
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"

	# Keep track of when the container was started.
	local started_at=$(docker_swarm inspect -f '{{ .State.StartedAt }}' test_container)

	# restart
	docker_swarm restart test_container

	# verify
	run docker_swarm ps -l
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"
	[ $(docker_swarm inspect -f '{{ .State.StartedAt }}' test_container) != "$started_at" ]
}
