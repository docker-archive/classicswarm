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
	[[ "${output}" == *"->80/tcp"* ]]
}

@test "docker-compose up - check bridge network" {
	# docker network connect --ip is introduced in docker 1.10, skip older version without --ip
	run docker network connect --help
	if [[ "${output}" != *"--ip"* ]]; then
		skip
	fi

	start_docker_with_busybox 2
	swarm_manage
	FILE=$TESTDATA/compose/simple_v2.yml

	docker-compose_swarm -f $FILE up -d

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  1 ]

	run docker_swarm inspect compose_service1_1
	[[ "${output}" == *"testn\""* ]]
}

function containerRunning() {
	local container="$1"
	local node="$2"
	run docker_swarm inspect "$container"
	[ "$status" -eq 0 ]
	[[ "${output}" == *"\"Name\": \"$node\""* ]]
	[[ "${output}" == *"\"Status\": \"running\""* ]]
}

@test "docker-compose up - reschedule" {
	start_docker_with_busybox 2
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]},${HOSTS[1]}
	FILE=$TESTDATA/compose/reschedule.yml

	docker-compose_swarm -f $FILE up -d
	
	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  3 ]

	# Make sure containers are running where they should.
	containerRunning "compose_service1_1" "node-0"
	containerRunning "compose_service2_1" "node-0"
	containerRunning "compose_service3_1" "node-0"

	# Get service1 swarm id
	swarm_id=$(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' compose_service1_1)

	# Stop node-0
	docker_host kill ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	# Wait for the container to be rescheduled
	# service1 should have been rescheduled from node-0 to node-1
	retry 5 1 containerRunning "compose_service1_1" "node-1"

	# Check swarm id didn't change for service1
	[[ "$swarm_id" == $(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' compose_service1_1) ]]

	run docker_swarm inspect "$swarm_id"
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm inspect compose_service2_1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# service_3 should still be on node-0 since the rescheduling policy was off.
	run docker_swarm inspect compose_service3_1
	[ "$status" -eq 1 ]

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]
}
