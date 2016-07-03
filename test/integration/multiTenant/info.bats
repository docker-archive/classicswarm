#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker info" {
	start_docker 2 --label foo=bar
	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 2"* ]]
	[[ "${output}" == *"└ Labels:"*"foo=bar"* ]]

}

@test "docker info - details" {
	# details in docker info were introduced in docker 1.10, skip older version without
	run docker info
	if [[ "${output}" != *"Paused:"* ]]; then
		skip
	fi

	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name test busybox sleep 100
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Running: 1"* ]]

	docker_swarm pause test
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Paused: 1"* ]]

	docker_swarm unpause test
	docker_swarm kill test
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Stopped: 1"* ]]
}