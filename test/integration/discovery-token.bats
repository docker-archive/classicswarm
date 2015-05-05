#!/usr/bin/env bats

load helpers

function token_cleanup() {
	curl -X DELETE https://discovery-stage.hub.docker.com/v1/clusters/$1
}

function setup() {
	start_docker 2
}

function teardown() {
	swarm_join_cleanup
	swarm_manage_cleanup
	stop_docker
}

@test "token discovery" {
	run swarm create
	[ "$status" -eq 0 ]
	[[ "$output" =~ ^[0-9a-f]{32}$ ]]
	TOKEN="$output"

	swarm_manage "token://$TOKEN"
	swarm_join   "token://$TOKEN"

	run docker_swarm info
	echo $output
	[[ "$output" == *"Nodes: 2"* ]]

	token_cleanup "$TOKEN"
}
