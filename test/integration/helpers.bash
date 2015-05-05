#!/bin/bash

# Root directory of the repository.
SWARM_ROOT=${SWARM_ROOT:-${BATS_TEST_DIRNAME}/../..}

# Path of the Swarm binary.
SWARM_BINARY=${SWARM_BINARY:-${SWARM_ROOT}/swarm}

# Docker image and version to use for integration tests.
DOCKER_IMAGE=${DOCKER_IMAGE:-dockerswarm/dind-master}
DOCKER_VERSION=${DOCKER_VERSION:-latest}
DOCKER_BINARY=${DOCKER_BINARY:-`command -v docker`}

# Host on which the manager will listen to (random port between 6000 and 7000).
SWARM_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 6000 ))

# Use a random base port (for engines) between 5000 and 6000.
BASE_PORT=$(( ( RANDOM % 1000 )  + 5000 ))

# Drivers to use for Docker engines the tests are going to create.
STORAGE_DRIVER=${STORAGE_DRIVER:-aufs}
EXEC_DRIVER=${EXEC_DRIVER:-native}

# Join an array with a given separator.
function join() {
	local IFS="$1"
	shift
	echo "$*"
}

# Run docker using the binary specified by $DOCKER_BINARY.
# This must ONLY be run on engines created with `start_docker`.
function docker() {
	"$DOCKER_BINARY" "$@"
}

# Communicate with Docker on the host machine.
# Should rarely use this.
function docker_host() {
	command docker "$@"
}

# Run the docker CLI against swarm.
function docker_swarm() {
	docker -H $SWARM_HOST "$@"
}

# Run the swarm binary. You must NOT fork this command (swarm foo &) as the PID
# ($!) will be the one of the subshell instead of swarm and you won't be able
# to kill it.
function swarm() {
	"$SWARM_BINARY" "$@"
}

# Retry a command $1 times until it succeeds. Wait $2 seconds between retries.
function retry() {
	local attempts=$1
	shift
	local delay=$1
	shift
	local i

	for ((i=0; i < attempts; i++)); do
		run "$@"
		if [[ "$status" -eq 0 ]] ; then
			return 0
		fi
		sleep $delay
	done

	echo "Command \"$@\" failed $attempts times. Output: $output"
	[[ false ]]
}

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	retry 10 1 docker -H $1 info
}

# Start the swarm manager in background.
function swarm_manage() {
	local discovery
	if [ $# -eq 0 ]; then
		discovery=`join , ${HOSTS[@]}`
	else
		discovery="$@"
	fi

	$SWARM_BINARY manage -H $SWARM_HOST $discovery &
	SWARM_PID=$!
	wait_until_reachable $SWARM_HOST
}

# Start swarm join for every engine with the discovery as parameter
function swarm_join() {
	local i=0
	for h in ${HOSTS[@]}; do
		echo "Swarm join #${i}: $h $@"
		$SWARM_BINARY join --addr=$h "$@" &
		SWARM_JOIN_PID[$i]=$!
		((++i))
	done
	retry 30 1 [ -n $(docker_swarm info | grep -q "Nodes: $i") ]
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

		# We have to manually call `hostname` since --hostname and --net cannot
		# be used together.
		DOCKER_CONTAINERS[$i]=$(
			docker_host run -d --name node-$i --privileged -it --net=host \
			${DOCKER_IMAGE}:${DOCKER_VERSION} \
			bash -c "\
				hostname node-$i && \
				docker -d -H 127.0.0.1:$port \
					--storage-driver=$STORAGE_DRIVER --exec-driver=$EXEC_DRIVER \
					`join ' ' $@` \
		")
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
		docker_host rm -f -v $id > /dev/null;
	done
}
