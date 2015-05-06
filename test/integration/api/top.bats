@test "docker top" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]
	# make sure container is running
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	run docker_swarm top test_container
	[ "$status" -eq 0 ]
	[[ "${lines[0]}" == *"UID"* ]]
	[[ "${lines[0]}" == *"CMD"* ]]
	[[ "${lines[1]}" == *"sleep 500"* ]]
}
