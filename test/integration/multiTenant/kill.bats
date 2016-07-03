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
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# kill
	docker_swarm kill test_container

	# verify
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=exited) ]
	[ $(docker_swarm inspect -f '{{ .State.ExitCode }}' test_container) -eq 137 ] 
}
