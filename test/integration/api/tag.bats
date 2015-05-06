#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker tag" {
	# Start one empty host and one with busybox to ensure swarm selects the
	# right one (and not one at random).
	start_docker 1
	start_docker_with_busybox 1
	swarm_manage

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
