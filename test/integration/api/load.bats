#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker load" {
	# temp file for saving image
	IMAGE_FILE=$(mktemp)

	# create a tar file
	docker_host pull busybox:latest
	docker_host save -o $IMAGE_FILE busybox:latest

	start_docker 2
	swarm_manage

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq  0 ]

	run docker_swarm load -i $IMAGE_FILE
	[ "$status" -eq 0 ]
	
	# check node0
	run docker -H  ${HOSTS[0]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	# check node1
	run docker -H  ${HOSTS[1]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	rm -f $IMAGE_FILE
}
