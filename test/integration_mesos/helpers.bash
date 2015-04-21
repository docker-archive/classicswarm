#!/bin/bash

# Root directory of the repository.
SWARM_ROOT=${SWARM_ROOT:-${BATS_TEST_DIRNAME}/../..}

# Host on which the manager will listen to (random port between 6000 and 7000).
SWARM_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 6000 ))

MESOS_CLUSTER_ENTRYPOINT=${MESOS_CLUSTER_ENTRYPOINT:-0.0.0.0:5050}

# Run the swarm binary.
function swarm() {
	godep go run "${SWARM_ROOT}/main.go" "$@"
}

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	local attempts=0
	local max_attempts=5
	until docker -H $1 info || [ $attempts -ge $max_attempts ]; do
		echo "Attempt to connect to $1 failed for the $((++attempts)) time" >&2
		sleep 0.5
	done
	[[ $attempts -lt $max_attempts ]]
}

# Run the docker CLI against swarm through Mesos.
function docker_swarm() {
	docker -H $SWARM_HOST "$@"
}

function swarm_manage() {
	${SWARM_ROOT}/swarm manage -c mesos -H $SWARM_HOST $MESOS_CLUSTER_ENTRYPOINT &
	SWARM_PID=$!
	wait_until_reachable $SWARM_HOST
}

function swarm_manage_cleanup() {
	kill $SWARM_PID
}

