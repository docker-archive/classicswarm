#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker port" {
	start_docker_with_busybox 2
	swarm_manage
	docker_swarm run -d -p 8000 --name test_container busybox sleep 500

	# make sure container is up
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# port verify
	run docker_swarm port test_container
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"8000"* ]]
}

@test "docker port parallel" {
	start_docker_with_busybox 2
	swarm_manage

	declare -a pids
	for i in 1 2 3; do
		# Use a non existing image to ensure that we pull at the same time
		docker_swarm run -d -p 8888 alpine:edge sleep 2 &
		pids[$i]=$!
	done

	# Wait for jobs in the background and check their exit status
	for pid in "${pids[@]}"; do
		wait $pid
		[ "$?" -eq 0 ]
	done
}
