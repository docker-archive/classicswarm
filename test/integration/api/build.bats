#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker build" {
	start_docker 2
	swarm_manage

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	docker_swarm build -t test $TESTDATA/build

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
}

@test "docker build with arg" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm build -t test_args --build-arg="greeting=Hello Args" $TESTDATA/build_with_args
	[ "$status" -eq 0 ]

	run docker_swarm run --rm test_args
	[ "$status" -eq 0 ]
	[[ "$output" == "Hello Args" ]]
}
