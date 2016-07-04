#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker stop" {
	start_docker_with_busybox 2
	swarm_manage
	# run 
	docker_swarm run -d --name test_container busybox sleep 500

	# make sure container is up before stop
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# stop
	docker_swarm stop test_container

	# verify
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=exited) ]
}
