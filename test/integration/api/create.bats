#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker create" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no contaienr exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# create
	docker_swarm create --name test_container busybox sleep 1000

	# verify, created container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
}

@test "docker create conflict" {
	start_docker_with_busybox 1
	swarm_manage

	id=$(docker_swarm create busybox)
	prefix=$(printf "$id" | cut -c 1-4)
	docker_swarm create --name "$prefix" busybox
	docker_swarm create --name "$id" busybox
	docker_swarm create --name test busybox
}