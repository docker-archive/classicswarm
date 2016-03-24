#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker volume ls" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no volume exist
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]

	# run
	docker_swarm run -d -v=/tmp busybox true

	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 2 ]

	docker_swarm run -d -v=/tmp busybox true

	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 3 ]
}

@test "docker volume inspect" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d -v=/tmp -e constraint:node==node-0 busybox true

	run docker_swarm volume ls -q
	[ "${#lines[@]}" -eq 1 ]
	[[ "${output}" == *"node-0/"* ]]

	id=${output}

	run docker_swarm volume inspect $id
	[[ "${output}" == *"\"Driver\": \"local\""* ]]

	diff <(docker_swarm volume inspect $id) <(docker -H ${HOSTS[0]} volume inspect ${id#node-0/})
}

@test "docker volume create" {
	start_docker 2
	swarm_manage

	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]

	docker_swarm volume create --name=test_volume
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 3 ]

	docker_swarm run -d -v=/tmp busybox true
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 4 ]

	run docker_swarm volume create --name=node-2/test_volume2
	[ "$status" -ne 0 ]

	docker_swarm volume create --name=node-0/test_volume2
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 5 ]
}

@test "docker volume rm" {
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
	
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]
}
