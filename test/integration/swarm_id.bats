#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "swarm id generation" {
	start_docker_with_busybox 1
	swarm_manage

	# Create a dummy container just so we interfere with the tests.
	# This one won't be used.
	docker_swarm run -d busybox true

	# Create a container and get its Swarm ID back.
	id=$(docker_swarm run -d busybox true)
	swarm_id=$(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' "$id")

	# Make sure we got a valid Swarm ID.
	[[ "${#swarm_id}" -eq 64 ]]
	[[ "$id" != "$swarm_id" ]]

	# API operations should work with Swarm IDs as well as Container IDs.
	[[ $(docker_swarm inspect -f "{{ .Id }}" "$swarm_id") == "$id" ]]

	# `docker ps` should be able to filter by Swarm ID using the label.
	[[ $(docker_swarm ps -a -q --no-trunc --filter="label=com.docker.swarm.id=$swarm_id") == "$id" ]]
}
