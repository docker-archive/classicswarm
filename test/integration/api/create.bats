@test "docker create" {
	start_docker 3
	swarm_manage

	# make sure no contaienr exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# create
	run docker_swarm create --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# verify, created container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
}
