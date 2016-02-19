#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker network ls" {
	start_docker 2
	swarm_manage

	run docker_swarm network ls
	[ "${#lines[@]}" -eq 7 ]
}

@test "docker network ls --filter type" {
	# docker network ls --filter type is introduced in docker 1.10, skip older version without --filter type
	run docker --version
	if [[ "${output}" != "Docker version 1.1"* ]]; then
		skip
	fi

	start_docker 2
	swarm_manage

	run docker_swarm network ls --filter type=builtin
	[ "${#lines[@]}" -eq 7 ]

	run docker_swarm network ls --filter type=custom
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm network ls --filter type=foo
	[ "$status" -ne 0 ]

	docker_swarm network create -d bridge test
	run docker_swarm network ls
	[ "${#lines[@]}" -eq 8 ]

	run docker_swarm network ls --filter type=custom
	[ "${#lines[@]}" -eq 2 ]
}

@test "docker network inspect" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d -e constraint:node==node-0 busybox sleep 100

	run docker_swarm network inspect bridge
	[ "$status" -ne 0 ]

	run docker_swarm network inspect node-0/bridge
	[[ "${output}" != *"\"Containers\": {}"* ]]

	diff <(docker_swarm network inspect node-0/bridge) <(docker -H ${HOSTS[0]} network inspect bridge)
}

@test "docker network create" {
	start_docker 2
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

@test "docker network rm" {
	start_docker_with_busybox 2
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

@test "docker network disconnect connect" {
	start_docker_with_busybox 2
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

@test "docker network connect --ip" {
	# docker network connect --ip is introduced in docker 1.10, skip older version without --ip
	run docker network connect --help
	if [[ "${output}" != *"--ip"* ]]; then
		skip
	fi

	start_docker_with_busybox 1
	swarm_manage

	docker_swarm network create -d bridge --subnet 10.0.0.0/24 testn

	run docker_swarm network inspect testn
	[[ "${output}" == *"\"Containers\": {}"* ]]

	# run
	docker_swarm run -d --name test_container  busybox sleep 100

	docker_swarm network connect --ip 10.0.0.42 testn test_container

	run docker_swarm inspect test_container
	[[ "${output}" == *"10.0.0.42"* ]]

	run docker_swarm network inspect testn
	[[ "${output}" != *"\"Containers\": {}"* ]]
}

@test "docker network connect --alias" {
	# docker network connect --alias is introduced in docker 1.10, skip older version without --alias
	run docker network connect --help
	if [[ "${output}" != *"--alias"* ]]; then
		skip
	fi

	start_docker_with_busybox 1
	swarm_manage

	docker_swarm network create -d bridge testn

	run docker_swarm network inspect testn
	[[ "${output}" == *"\"Containers\": {}"* ]]

	# run
	docker_swarm run -d --name test_container  busybox sleep 100

	docker_swarm network connect --alias testa testn test_container

	run docker_swarm inspect test_container
	[[ "${output}" == *"testa"* ]]

	run docker_swarm network inspect testn
	[[ "${output}" != *"\"Containers\": {}"* ]]
}

@test "docker run --net <node>/<network>" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm network create -d bridge node-1/testn

	docker_swarm run -d --net node-1/testn --name test_container busybox sleep 100

	run docker_swarm network inspect testn
	[[ "${output}" != *"\"Containers\": {}"* ]]
}
