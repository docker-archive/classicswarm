@test "docker build" {
	start_docker 3
	swarm_manage

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	run docker_swarm build -t test $BATS_TEST_DIRNAME/testdata/build
	[ "$status" -eq 0 ]

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
}
