#!/bin/bash

# Root directory of the repository.
SWARM_ROOT=${SWARM_ROOT:-${BATS_TEST_DIRNAME}/../..}

# Docker image and version to use for integration tests.
DOCKER_IMAGE=${DOCKER_IMAGE:-aluzzardi/docker}
DOCKER_VERSION=${DOCKER_VERSION:-1.5}

# Host on which the manager will listen to (random port between 6000 and 7000).
SWARM_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 6000 ))

# Use a random base port (for engines) between 5000 and 6000.
BASE_PORT=$(( ( RANDOM % 1000 )  + 5000 ))

# Join an array with a given separator.
function join() {
	local IFS="$1"
	shift
	echo "$*"
}

# Run the swarm binary.
function swarm() {
	godep go run "${SWARM_ROOT}/main.go" "$@"
}

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	local attempts=0
	local max_attempts=5
	until docker -H $1 info || [ $attempts -ge $max_attempts ]; do
		echo "Attempt to connect to ${HOSTS[$i]} failed for the $((++attempts)) time" >&2
		sleep 0.5
	done
	[[ $attempts -lt $max_attempts ]]
}

# Start the swarm manager in background.
function swarm_manage() {
	local discovery
	if [ $# -eq 0 ]; then
		discovery=`join , ${HOSTS[@]}`
	else
		discovery="$@"
	fi

	swarm manage -H $SWARM_HOST $discovery &
	SWARM_PID=$!
	wait_until_reachable $SWARM_HOST
}

# Start swarm join for every engine with the discovery as parameter
function swarm_join() {
	local i=0
	for h in ${HOSTS[@]}; do
		echo "Swarm join #${i}: $h $@"
		swarm join --addr=$h "$@" &
		SWARM_JOIN_PID[$i]=$!
		((++i))
	done
	wait_until_swarm_joined $i
}

# Wait until a swarm instance joins the cluster.
# Parameter $1 is number of nodes to check.
function wait_until_swarm_joined {
	local attempts=0
	local max_attempts=10

	until [ $attempts -ge $max_attempts ]; do
		run docker -H $SWARM_HOST info
		if [[ "${lines[3]}" == *"Nodes: $1"* ]]; then
			break
		fi 
		echo "Checking if joined successfully for the $((++attempts)) time" >&2
		sleep 1
	done
	[[ $attempts -lt $max_attempts ]]
}

# Stops the manager.
function swarm_manage_cleanup() {
	kill $SWARM_PID
}

# Clean up Swarm join processes
function swarm_join_cleanup() {
	for pid in ${SWARM_JOIN_PID[@]}; do
		kill $pid
	done
}

# Run the docker CLI against swarm.
function docker_swarm() {
	docker -H $SWARM_HOST "$@"
}

# Start N docker engines.
function start_docker() {
	local current=${#DOCKER_CONTAINERS[@]}
	local instances="$1"
	shift
	local i

	# Start the engines.
	for ((i=current; i < (current + instances); i++)); do
		local port=$(($BASE_PORT + $i))
		HOSTS[$i]=127.0.0.1:$port
		DOCKER_CONTAINERS[$i]=$(docker run -d --name node-$i -h node-$i --privileged -p 127.0.0.1:$port:$port -it ${DOCKER_IMAGE}:${DOCKER_VERSION} docker -d -H 0.0.0.0:$port "$@")
	done

	# Wait for the engines to be reachable.
	for ((i=current; i < (current + instances); i++)); do
		wait_until_reachable ${HOSTS[$i]}
	done
}

# Stop all engines.
function stop_docker() {
	for id in ${DOCKER_CONTAINERS[@]}; do
		echo "Stopping $id"
		docker rm -f -v $id > /dev/null;
	done
}
