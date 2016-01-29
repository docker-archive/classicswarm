#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "resource limitation: memory" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm run -m 1000G busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"no resources available to schedule container"* ]]

	# The number of running containers should be still 0.
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 2"* ]]

	docker_swarm run --name container_test -e constraint:node==node-0 -m 20m busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 20 MiB"* ]]
	[[ "${output}" == *"Reserved Memory: 0 B"* ]]

	docker_swarm run --name container_test2 -e constraint:node==node-0 -m 22m busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 42 MiB"* ]]
	[[ "${output}" == *"Reserved Memory: 0 B"* ]]

	docker_swarm run --name container_test3 -e constraint:node==node-1 -m 18m busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 42 MiB"* ]]
	[[ "${output}" == *"Reserved Memory: 18 MiB"* ]]

	docker_swarm rm container_test

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 22 MiB"* ]]
	[[ "${output}" == *"Reserved Memory: 18 MiB"* ]]

	docker_swarm rm container_test2

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 0 B"* ]]
	[[ "${output}" == *"Reserved Memory: 18 MiB"* ]]

}

@test "resource limitation: cpu" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm run -c 10240 busybox sh
	[ "$status" -ne 0 ]
	[[ "${output}" == *"no resources available to schedule container"* ]]

	# The number of running containers should be still 0.
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 2"* ]]

	docker_swarm run --name container_test -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 1"* ]]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]

	docker_swarm rm container_test

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]
}

@test "strategy spread" {
	start_docker_with_busybox 2
	swarm_manage --strategy spread ${HOSTS[0]},${HOSTS[1]}

	docker_swarm run --name container_test1 -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 1"* ]]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]

	docker_swarm run --name container_test2 -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 1"* ]]
	[[ "${output}" != *"Reserved CPUs: 0"* ]]
}

@test "strategy binpack" {
	start_docker_with_busybox 2
	swarm_manage --strategy binpack ${HOSTS[0]},${HOSTS[1]}

	docker_swarm run --name container_test1 -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 1"* ]]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]

	docker_swarm run --name container_test2 -c 1 busybox sh
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved CPUs: 2"* ]]
	[[ "${output}" == *"Reserved CPUs: 0"* ]]
}
