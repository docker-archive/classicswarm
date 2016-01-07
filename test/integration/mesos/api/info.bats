#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker info" {
	start_docker 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *'Offers: 2'* ]]
}