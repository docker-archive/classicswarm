#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker version" {
	# FIXME: No reason here to start docker.
	start_docker 1
	swarm_manage

	# version
	run docker_swarm version
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 8 ]

	# verify
	client_reg='^Client version: [0-9]+\.[0-9]+\.[0-9]+.*$'
	server_reg='^Server version: swarm/[0-9]+\.[0-9]+\.[0-9]+.*$'
	[[ "${lines[0]}" =~ $client_reg ]]
	[[ "${lines[5]}" =~ $server_reg ]]
}
