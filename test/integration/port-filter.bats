#!/usr/bin/env bats

load helpers

function teardown() {
	stop_manager
	stop_docker
}

@test "docker should filter port in host mode correctly" {
	start_docker 2
	start_manager

	#
	# Use busybox to save image pulling time for integration test.
	# Running the first 2 containers, it should be fine.
	#
	run docker_swarm run -d --expose=80 --net=host busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d --expose=80 --net=host busybox sh
	[ "$status" -eq 0 ]

	#
	# When trying to start the 3rd one, it should be error finding port 80.
	#
	run docker_swarm run -d --expose=80 --net=host busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"unable to find a node with port 80/tcp available in the Host mode"* ]]

	#
	# And the number of running containers should be still 2.
	#
	run docker_swarm ps -n 2
	[ "${#lines[@]}" -eq  3 ]
}
