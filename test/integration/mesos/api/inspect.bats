#!/usr/bin/env bats

load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "docker inspect" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage_mesos
	# run container
	docker_swarm run -d -m 20m -e TEST=true --name test_container busybox sleep 500

	# make sure container exsists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	# inspect and verify 
	run docker_swarm inspect test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == *"NetworkSettings"* ]]
	[[ "${output}" == *"TEST=true"* ]]
	# the specific information of swarm node
	[[ "${output}" == *'"Node": {'* ]]
	[[ "${output}" == *'"Name": "node-'* ]]
}

