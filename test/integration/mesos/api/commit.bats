#!/usr/bin/env bats

load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker commit" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage_mesos

	docker_swarm run -d -m 20m --name test_container busybox sleep 500

	# make sure container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# no coming name before commit 
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" != *"commit_image_busybox"* ]]

	# commit container
	docker_swarm commit test_container commit_image_busybox

	# verify after commit 
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"commit_image_busybox"* ]]
}
