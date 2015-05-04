#!/usr/bin/env bats

load helpers

function token_cleanup() {
	curl -X DELETE https://discovery-stage.hub.docker.com/v1/clusters/$1
}

function teardown() {
	swarm_join_cleanup
	swarm_manage_cleanup
	stop_docker
}

@test "token discovery should be working properly" {
	start_docker 2

	TOKEN=$(swarm create)
	[ "$status" -eq 0 ]
	[[ ${TOKEN} =~ ^[0-9a-f]{32}$ ]]

	swarm_manage token://$TOKEN
	swarm_join   token://$TOKEN

	run docker_swarm info
	[[ "${lines[3]}" == *"Nodes: 2"* ]]

	token_cleanup $TOKEN
}
