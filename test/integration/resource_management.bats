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
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 1 ]

	# Node is still 1
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 1"* ]]

	docker_swarm run --name container_test -m 20m busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 20 MiB"* ]]

	docker_swarm run --name container_test2 -m 22m busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 42 MiB"* ]]

	docker_swarm rm container_test

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 22 MiB"* ]]

	docker_swarm rm container_test2

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 0 B"* ]]

}

@test "resource limitation: cpu" {
	start_docker_with_busybox 1
	swarm_manage

	run docker_swarm run -c 10240 busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"no resources available to schedule container"* ]]

	# The number of running containers should be still 0.
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 1 ]

	# Node is still 1
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 1"* ]]

	docker_swarm run --name container_test -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 1"* ]]

	docker_swarm rm container_test

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]
}
