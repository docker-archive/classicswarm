#!/usr/bin/env bats

load compose_helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker-compose up" {
	start_docker_with_busybox 2
	swarm_manage
	FILE=$TESTDATA/compose/simple.yml

	docker-compose_swarm -f $FILE up -d

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]
}

@test "docker-compose up - check memory swappiness" {
	start_docker_with_busybox 2
	swarm_manage
	FILE=$TESTDATA/compose/simple.yml

	docker-compose_swarm -f $FILE up -d

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]

	run docker_swarm inspect compose_service1_1
	# check memory-swappiness
	[[ "${output}" == *"\"MemorySwappiness\": -1"* ]]
}

@test "docker-compose up - check port" {
	start_docker_with_busybox 2
	swarm_manage
	FILE=$TESTDATA/compose/simple.yml

	docker-compose_swarm -f $FILE up -d

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]

	run docker_swarm ps
	# check memory-swappiness
echo $output
	[[ "${output}" == *"->80/tcp"* ]]
}
