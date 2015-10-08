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

@test "docker network inspect" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d -e constraint:node==node-0 busybox sleep 100

	run docker_swarm network inspect bridge
	[ "$status" -ne 0 ]

	run docker_swarm network inspect node-0/bridge
	[ "${#lines[@]}" -eq 13 ]
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

@test "docker volume rm" {
skip
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm volume rm test_volume
	[ "$status" -ne 0 ]

	docker_swarm run -d --name=test_container -v=/tmp busybox true
	
	run docker_swarm volume ls -q
	volume=${output}
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm volume rm $volume
	[ "$status" -ne 0 ]

	docker_swarm rm test_container
	
	run docker_swarm volume rm $volume
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	
	run docker_swarm volume
	echo $output
	[ "${#lines[@]}" -eq 1 ]
}
