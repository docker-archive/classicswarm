@test "docker history" {
	start_docker 3
	swarm_manage

	# pull busybox image
	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure the image of busybox exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]

	# history
	run docker_swarm history busybox
	[ "$status" -eq 0 ]
	[[ "${lines[0]}" == *"CREATED BY"* ]]
}
