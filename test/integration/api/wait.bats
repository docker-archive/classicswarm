@test "docker wait" {
	start_docker 3
	swarm_manage

	# run after 1 seconds, test_container will exit
	run docker_swarm run -d --name test_container busybox sleep 1
	[ "$status" -eq 0 ]

	# make sure container exists and is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
	[[ "${lines[1]}" ==  *"Up"* ]]

	# wait until exist(after 1 seconds)
	run timeout 5 docker -H $SWARM_HOST wait test_container

	[ "${#lines[@]}" -eq 1 ]
	[[ "${output}" == "0" ]]
}
