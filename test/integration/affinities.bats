#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "container affinty" {
	start_docker 2
	swarm_manage

	run docker_swarm run --name c1 -e constraint:node==node-0 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c2 -e affinity:container==c1 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c3 -e affinity:container!=c1 -d busybox:latest sh
	[ "$status" -eq 0 ]

	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}

@test "label affinity" {
	start_docker 2
	swarm_manage

	run docker_swarm run --name c1 --label test.label=true -e constraint:node==node-0 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c2 -e affinity:test.label==true -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c3 -e affinity:test.label!=true -d busybox:latest sh
	[ "$status" -eq 0 ]

	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}
