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
	# the coming image of tag_busybox not exsit
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" != *"tag_busybox"* ]]

	# tag image
	docker_swarm tag busybox tag_busybox:test

	# verify
	run docker_swarm images tag_busybox
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tag_busybox"* ]]
}

@test "docker tag multi-nodes with same image" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure busybox exists
	# and tag_busybox not exists
	run docker_swarm images
	[ "${#lines[@]}" -ge 2 ]
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" != *"tag_busybox"* ]]

	# tag image
	docker_swarm tag busybox tag_busybox:test

	# verify
	# change the way to verify tagged image on each node after image deduplication
	run docker_swarm images --filter node=node-0
	[[ $(echo ${output} | grep -o "tag_busybox" | wc -l) == 1 ]]

	run docker_swarm images --filter node=node-1
	[[ $(echo ${output} | grep -o "tag_busybox" | wc -l) == 1 ]]
}
