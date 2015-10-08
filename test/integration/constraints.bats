#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "node constraint" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm run --name c1 -e constraint:node==node-0 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c2 -e constraint:node==node-1 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c3 -e constraint:node==node-1 -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c4 --label 'com.docker.swarm.constraints=["node==node-1"]' -d busybox:latest sh
	[ "$status" -eq 0 ]
	
	run docker_swarm inspect c1
	# FIXME: This will help debugging the failing test.
	echo $output
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm inspect c4
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}

@test "soft constraint" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run --name c1 -e constraint:storagedriver==~not_exist -e constraint:node==node-0 -d busybox:latest sh
	docker_swarm run --name c2 -e constraint:storagedriver==~not_exist -e constraint:node==node-0 -d busybox:latest sh

	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]
}

@test "label constraints" {
	start_docker_with_busybox 1 --label foo=a
	start_docker_with_busybox 1 --label foo=b
	swarm_manage

	run docker_swarm run --name c1 -e constraint:foo==a -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c2 -e constraint:foo==b -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c3 -e constraint:foo==b -d busybox:latest sh
	[ "$status" -eq 0 ]
	run docker_swarm run --name c4 --label 'com.docker.swarm.constraints=["foo==b"]' -d busybox:latest sh
	[ "$status" -eq 0 ]

	run docker_swarm inspect c1
	# FIXME: This will help debugging the failing test.
	echo $output
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]

	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm inspect c4
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}
