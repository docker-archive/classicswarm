#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker load" {
	# pull the image only if not available on the host and save it somewhere.
	[ "$(docker_host images -q busybox)" ] || docker_host pull busybox
	IMAGE_FILE=$(mktemp)
	docker_host save -o $IMAGE_FILE busybox:latest

	start_docker 2
	swarm_manage

	# ensure we start from a clean cluster.
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq  0 ]

	docker_swarm load -i $IMAGE_FILE

	# and now swarm should have cought the image just loaded.
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge  1 ]
	
	# check node0
	run docker -H ${HOSTS[0]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	# check node1
	run docker -H ${HOSTS[1]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	rm -f $IMAGE_FILE
}
