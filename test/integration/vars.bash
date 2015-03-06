#!/bin/bash

# Root directory of the repository.
SWARM_ROOT=${BATS_TEST_DIRNAME}/../..

# Docker image and version to use for integration tests.
DOCKER_IMAGE=${DOCKER_IMAGE:-aluzzardi/docker}
DOCKER_VERSION=${DOCKER_VERSION:-1.5}

# Host on which the manager will listen to (random port between 6000 and 7000).
SWARM_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 6000 ))

# Use a random base port (for engines) between 5000 and 6000.
BASE_PORT=$(( ( RANDOM % 1000 )  + 5000 ))

# Join an array with a given separator.
function join { local IFS="$1"; shift; echo "$*"; }

# Run the swarm binary.
function swarm() {
	${SWARM_ROOT}/swarm $@
}

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	local attempts=0
	until docker -H $1 info &> /dev/null || [ $attempts -ge 10 ]; do
		echo "Attempt to connect to ${HOSTS[$i]} failed for the $((++attempts)) time" >&2
		sleep 0.5
	done
}

# Start the swarm manager in background.
function start_manager() {
	${SWARM_ROOT}/swarm manage -H $SWARM_HOST $@ `join , ${HOSTS[@]}` &
	SWARM_PID=$!
	wait_until_reachable $SWARM_HOST
}

# Stops the manager.
function stop_manager() {
	kill $SWARM_PID
}

# Run the docker CLI against swarm.
function docker_swarm() {
	docker -H $SWARM_HOST $@
}

# Start N docker engines.
function start_docker() {
	local current=${#DOCKER_CONTAINERS[@]}
	local instances="$1"
	shift
	local args="$@"

	# Start the engines.
	for i in `seq $current $((instances - 1))`; do
		local port=$(($BASE_PORT + $i))
		HOSTS[$i]=127.0.0.1:$port
		DOCKER_CONTAINERS[$i]=$(docker run -d --name node-$i -h node-$i --privileged -p 127.0.0.1:$port:$port -it ${DOCKER_IMAGE}:${DOCKER_VERSION} docker -d -H 0.0.0.0:$port $args)
	done

	# Wait for the engines to be reachable.
	for i in `seq $current $((instances - 1))`; do
		wait_until_reachable ${HOSTS[$i]}
	done
}

# Stop all engines.
function stop_docker() {
	for id in ${DOCKER_CONTAINERS[@]}; do
		echo "Stopping $id"
		docker rm -f $id > /dev/null;
	done
}
