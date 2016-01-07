#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker ps" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -m 20m -d --name test_container busybox sleep 100

	# verify, container is running
	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq 1 ]

	run docker -H ${HOSTS[0]} run -d --name test_container2 busybox sleep 100

	# verify, container is running
	run docker -H  ${HOSTS[0]} ps -q --filter=name=test_container2 --filter=status=running
	[ "${#lines[@]}" -eq 1 ]

	# check we only the swarm containers are displayed
	run docker_swarm ps -q
	[ "${#lines[@]}" -eq 1 ]
}