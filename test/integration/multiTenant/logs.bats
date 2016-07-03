#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker logs" {
	start_docker_with_busybox 2
	swarm_manage

	# run a container with echo command
	docker_swarm run -d --name test_container busybox /bin/sh -c "echo hello world; echo hello docker; echo hello swarm"

	# make sure container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# verify
	run docker_swarm logs test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[0]}" ==  *"hello world"* ]]
	[[ "${lines[1]}" ==  *"hello docker"* ]]
	[[ "${lines[2]}" ==  *"hello swarm"* ]]
}

@test "docker logs unhealthy node" {
	start_docker_with_busybox 1
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]}

	# run a container with echo command
	docker_swarm run -d --name test_container busybox /bin/sh -c "echo hello world; echo hello docker; echo hello swarm"

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"


	# verify
	run docker_swarm logs test_container
	[ "$status" -eq 1 ]
	[[ "${output}" == *" running on unhealthy node"* ]]
}
