#!/usr/bin/env bats

load mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "docker info" {
	start_docker 1
	start_mesos
	swarm_manage_mesos
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *'Offers'* ]]
}

@test "docker run no resources" {
	start_docker 1
	start_mesos
	swarm_manage_mesos
	run docker_swarm run -d busybox ls
	[ "$status" -ne 0 ]
	[[ "${output}" == *'Task uses no resources'* ]]
}

@test "docker run" {
	start_docker_with_busybox 1
	start_mesos
	swarm_manage_mesos
	docker_swarm run -d -m 20m busybox ls
	docker_swarm run -d -m 20m busybox ls
}


