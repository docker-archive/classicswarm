#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "containerslots filter" {
	start_docker_with_busybox 2 --label containerslots=2
	swarm_manage

	# Use busybox to save image pulling time for integration test.
	# Running the first 4 containers, it should be fine.
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]

	# When trying to start the 5th one, it should be error finding a node with free slots.
	run docker_swarm run -d -t busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Unable to find a node that satisfies the following conditions"* ]]
	[[ "${lines[1]}" == *"free slots"* ]]

	# And the number of running containers should be still 4.
	run docker_swarm ps
	[ "${#lines[@]}" -eq 5 ]
}

@test "containerslots without existing label" {
	start_docker_with_busybox 2
	swarm_manage

	# Use busybox to save image pulling time for integration test.
	# Running more than 5 containers, it should be fine.
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]

	# And the number of running containers should be 5.
	run docker_swarm ps
	[ "${#lines[@]}" -eq 6 ]
}

@test "containerslots with invalid label" {
	start_docker_with_busybox 2 --label containerslots="foo"
	swarm_manage

	# Use busybox to save image pulling time for integration test.
	# Running more than 5 containers, it should be fine.
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -d -t busybox sh
	[ "$status" -eq 0 ]

	# And the number of running containers should be 5.
	run docker_swarm ps
	[ "${#lines[@]}" -eq 6 ]
}