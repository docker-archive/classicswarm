#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker rmi" {
	# Start one empty host and two with busybox to ensure swarm selects the
	# right ones and rmi doesn't fail if one host doesn't have the image.
	start_docker 1
	start_docker_with_busybox 2
	swarm_manage

	# make sure image exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]

	# verify the nodes: the first one shouldn't have the image while the other
	# two yes.
	run docker -H ${HOSTS[0]} images
	[ "$status" -eq 0 ]
	[[ "${output}" != *"busybox"* ]]
	run docker -H ${HOSTS[1]} images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]
	run docker -H ${HOSTS[1]} images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]


	# wipe busybox.
	docker_swarm rmi busybox

	# swarm verify
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	# verify the image was actually removed from every node.
	for host in ${HOSTS[@]}; do
		run docker -H $host images -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done
}
