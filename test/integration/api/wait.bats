#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker wait" {
	start_docker_with_busybox 2
	swarm_manage

	# run after 1 seconds, test_container will exit
	docker_swarm run -d --name test_container busybox sleep 1

	# wait until exist(after 1 seconds)
	run timeout 5 docker -H $SWARM_HOST wait test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == "0" ]]
}
