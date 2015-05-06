@test "docker tag" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure the image of busybox exists 
	# the comming image of tag_busybox not exsit
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" != *"tag_busybox"* ]]

	# tag image
	run docker_swarm tag busybox tag_busybox:test
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm images tag_busybox
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tag_busybox"* ]]
}
