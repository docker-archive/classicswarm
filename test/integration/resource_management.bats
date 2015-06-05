#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "resource limitation: memory" {
	start_docker_with_busybox 1
	swarm_manage

	run docker_swarm run -m 1000G busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"no resources available to schedule container"* ]]

	# The number of running containers should be still 0.
	run docker_swarm ps -n 2
	[ "${#lines[@]}" -eq 1 ]

	# Node is still 1
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 1"* ]]
}

@test "resource limitation: cpu" {
	start_docker_with_busybox 1
	swarm_manage

	run docker_swarm run -c 10240 busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"no resources available to schedule container"* ]]

	# The number of running containers should be still 0.
	run docker_swarm ps -n 2
	[ "${#lines[@]}" -eq 1 ]

	# Node is still 1
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 1"* ]]
}
