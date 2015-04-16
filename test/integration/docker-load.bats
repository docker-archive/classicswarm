#!/usr/bin/env bats
load helpers

# temp file for saving image
IMAGE_FILE=$(mktemp)

function teardown() {
	stop_docker
	swarm_manage_cleanup
	rm -f $IMAGE_FILE
}

@test "docker load should return success,every node should load the image" {
	# create a tar file
	docker pull busybox:latest
	docker save -o $IMAGE_FILE busybox:latest

	start_docker 2
	swarm_manage

	run docker_swarm load -i $IMAGE_FILE
	[ "$status" -eq 0 ]
    
	run docker -H  ${HOSTS[0]} images
	[ "${#lines[@]}" -eq  2 ]

	run docker -H  ${HOSTS[1]} images
	[ "${#lines[@]}" -eq  2 ]
}

