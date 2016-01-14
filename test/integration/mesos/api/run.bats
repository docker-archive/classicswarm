#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker run with wrong user" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental --cluster-opt mesos.user=test_wrong_user 127.0.0.1:$MESOS_MASTER_PORT

	# run
	run docker_swarm run -m 20m -d --name test_container busybox sleep 100

	# error check
	[ "$status" -ne 0 ]
	[[ "${output}" == *"please verify your SWARM_MESOS_USER is correctly set"* ]]
}

@test "mesos - docker run" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -m 20m -d --name test_container busybox sleep 100

	# verify, container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# error check
	run docker_swarm run -m 20m -d 4e8aa3148a132f19ec560952231c4d39522043994df7d2dc239942c0f9424ebd
	[[ "${output}" == *"cannot specify 64-byte hexadecimal strings"* ]]
}

@test "mesos - docker run short lived" {
	start_docker_with_busybox 1
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# run
	run docker_swarm run -m 20m busybox true

	# make sure the container was started
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 1 ]
}

@test "mesos - docker run no resources" {
	start_docker 1
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	run docker_swarm run -d busybox ls
	[ "$status" -ne 0 ]
	[[ "${output}" == *'resources constraints (-c and/or -m) are required by mesos'* ]]
}

@test "mesos - docker run with long pull" {
	start_docker 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental --cluster-opt mesos.tasktimeout=1s 127.0.0.1:$MESOS_MASTER_PORT

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -m 20m -d --name test_container busybox sleep 100

	# verify, container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]
}
