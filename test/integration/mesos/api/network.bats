#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker network ls" {
	start_docker 2
	start_mesos
	swarm_manage

	run docker_swarm network ls
	echo $output
	[ "${#lines[@]}" -eq 7 ]
}

@test "mesos - docker network inspect" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage

	# run
	docker_swarm run -d -e constraint:node==node-0 busybox sleep 100

	run docker_swarm network inspect bridge
	[ "$status" -ne 0 ]

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" != *"\"Containers\": {}"* ]]

	diff <(docker_swarm network inspect node-0/bridge) <(docker -H ${HOSTS[0]} network inspect bridge)
}

@test "mesos - docker network create" {
	start_docker 2
	start_mesos
	swarm_manage

	run docker_swarm network ls
	[ "${#lines[@]}" -eq 7 ]

	docker_swarm network create -d bridge test1
	run docker_swarm network ls
	[ "${#lines[@]}" -eq 8 ]

	docker_swarm network create -d bridge node-1/test2
	run docker_swarm network ls
	[ "${#lines[@]}" -eq 9 ]

	run docker_swarm network create -d bridge node-2/test3
	[ "$status" -ne 0 ]
}

@test "mesos - docker network rm" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage

	run docker_swarm network rm test_network
	[ "$status" -ne 0 ]

	run docker_swarm network rm bridge
	[ "$status" -ne 0 ]

	docker_swarm network create -d bridge node-0/test
	run docker_swarm network ls
	[ "${#lines[@]}" -eq 8 ]

	docker_swarm network rm node-0/test
	run docker_swarm network ls
	[ "${#lines[@]}" -eq 7 ]
}

@test "mesos - docker network disconnect connect" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage

	# run
	docker_swarm run -d --name test_container -e constraint:node==node-0 busybox sleep 100

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" != *"\"Containers\": {}"* ]]

	docker_swarm network disconnect node-0/bridge test_container

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" == *"\"Containers\": {}"* ]]

	docker_swarm network connect node-0/bridge test_container

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" != *"\"Containers\": {}"* ]]

	docker_swarm rm -f test_container

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" == *"\"Containers\": {}"* ]]
}
