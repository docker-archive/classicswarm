#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "build with impossible node constraint" {
	start_docker 2
	swarm_manage

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	run docker_swarm build --build-arg="constraint:node==node-9" $TESTDATA/build
	[ "$status" -eq 1 ]
	[[ "${lines[1]}" == *"Unable to find a node that satisfies the following conditions"* ]]
	[[ "${lines[2]}" == *"[node==node-9]"* ]]

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
}

@test "build with node constraint and buildarg" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm build -t test_args --build-arg="greeting=Hello Args" --build-arg="constraint:node==node-1" $TESTDATA/build_with_args
	[ "$status" -eq 0 ]

	run docker_swarm run --name c1 test_args
	[ "$status" -eq 0 ]
	[[ "$output" == "Hello Args" ]]

	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}
