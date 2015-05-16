#!/usr/bin/env bats

load discovery_helpers

TOKEN=""
DISCOVERY=""

function token_cleanup() {
	[ -z "$TOKEN" ] && return
	echo "Removing $TOKEN"
	curl -X DELETE "https://discovery-stage.hub.docker.com/v1/clusters/$TOKEN"
}

function setup() {
	TOKEN=$(swarm create)
	[[ "$TOKEN" =~ ^[0-9a-f]{32}$ ]]
	DISCOVERY="token://$TOKEN"
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	token_cleanup
}

@test "token discovery: recover engines" {
	# The goal of this test is to ensure swarm can see engines that joined
	# while the manager was stopped.

	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"

	# Then, start a manager and ensure it sees all the engines.
	swarm_manage "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}

@test "token discovery: watch for changes" {
	# The goal of this test is to ensure swarm can see new nodes as they join
	# the cluster.

	# Start a manager with no engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add engines to the cluster and make sure it's picked up by swarm.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}
