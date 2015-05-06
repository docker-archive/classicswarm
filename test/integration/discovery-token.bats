#!/usr/bin/env bats

load helpers

TOKEN=""

function token_cleanup() {
	[ -z "$TOKEN" ] && return
	echo "Removing $TOKEN"
	curl -X DELETE "https://discovery-stage.hub.docker.com/v1/clusters/$TOKEN"
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	token_cleanup
}

@test "token discovery" {
	# Create a cluster and validate the token.
	run swarm create
	[ "$status" -eq 0 ]
	[[ "$output" =~ ^[0-9a-f]{32}$ ]]
	TOKEN="$output"

	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "token://$TOKEN"

	# Start a manager and ensure it sees all the engines.
	swarm_manage "token://$TOKEN"
	check_swarm_nodes

	# Add another engine to the cluster and make sure it's picked up by swarm.
	start_docker 1
	swarm_join "token://$TOKEN"
	retry 10 1 check_swarm_nodes
}
