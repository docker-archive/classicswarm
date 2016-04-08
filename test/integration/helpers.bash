#!/bin/bash

# Root directory of integration tests.
INTEGRATION_ROOT=$(dirname "$(readlink -f "$BASH_SOURCE")")

# Test data path.
TESTDATA="${INTEGRATION_ROOT}/testdata"

# Root directory of the repository.
SWARM_ROOT=${SWARM_ROOT:-$(cd "$INTEGRATION_ROOT/../.."; pwd -P)}

# Path of the Swarm binary.
SWARM_BINARY=${SWARM_BINARY:-${SWARM_ROOT}/swarm}

# Docker image and version to use for integration tests.
DOCKER_IMAGE=${DOCKER_IMAGE:-dockerswarm/dind-master}
DOCKER_VERSION=${DOCKER_VERSION:-latest}
DOCKER_BINARY=${DOCKER_BINARY:-`command -v docker`}
DOCKER_COMPOSE_VERSION=${DOCKER_COMPOSE_VERSION:-1.6.2}

# Port on which the manager will listen to (random port between 6000 and 7000).
SWARM_BASE_PORT=$(( ( RANDOM % 1000 )  + 6000 ))

# Use a random base port (for engines) between 5000 and 6000.
BASE_PORT=$(( ( RANDOM % 1000 )  + 5000 ))

# Drivers to use for Docker engines the tests are going to create.
STORAGE_DRIVER=${STORAGE_DRIVER:-aufs}

BUSYBOX_IMAGE="$BATS_TMPDIR/busybox.tgz"

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
	docker -H ${SWARM_HOSTS[0]} "$@"
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
	false
}

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	retry 15 1 docker -H $1 info
}

# Returns true if all nodes have been added to swarm. Note some may be in pending state.
function discovery_check_swarm_info() {
	local total="$1"
	[ -z "$total" ] && total="${#HOSTS[@]}"
	local host="$2"
	[ -z "$host" ] && host="${SWARM_HOSTS[0]}"

	eval "docker -H $host info | grep -q -e \"Nodes: $total\" -e \"Offers: $total\""
}

# Return true if all nodes has been validated
function nodes_validated() {
	# Nodes are not in Pending state
	[[ $(docker_swarm info | grep -c "Status: Pending") -eq 0 ]]
}

function swarm_manage() {
	local i=${#SWARM_MANAGE_PID[@]}

	swarm_manage_no_wait "$@"

	# Wait for nodes to be discovered
	retry 10 1 discovery_check_swarm_info "${#HOSTS[@]}" "${SWARM_HOSTS[$i]}"

	# All nodes passes pending state
	retry 15 1 nodes_validated
}

# Start the swarm manager in background.
function swarm_manage_no_wait() {
	local discovery
	if [ $# -eq 0 ]; then
		discovery=`join , ${HOSTS[@]}`
	else
		discovery="$@"
	fi

	local i=${#SWARM_MANAGE_PID[@]}
	local port=$(($SWARM_BASE_PORT + $i))
	local host=127.0.0.1:$port

	"$SWARM_BINARY" -l debug -experimental manage -H "$host" --heartbeat=1s $discovery &
	SWARM_MANAGE_PID[$i]=$!
	SWARM_HOSTS[$i]=$host

	# Wait for the Manager to be reachable
	wait_until_reachable "$host"
}

# swarm join every engine created with `start_docker`.
#
# It will wait until all nodes are visible in discovery (`swarm list`) before
# returning and will fail if that's not the case after a certain time.
#
# It can be called multiple times and will only join new engines started with
# `start_docker` since the last `swarm_join` call.
function swarm_join() {
	local current=${#SWARM_JOIN_PID[@]}
	local nodes=${#HOSTS[@]}
	local addr="$1"
	shift

	# Start the engines.
	local i
	for ((i=current; i < nodes; i++)); do
		local h="${HOSTS[$i]}"
		echo "Swarm join #${i}: $h $addr"
		"$SWARM_BINARY" -l debug join --heartbeat=1s --ttl=10s --advertise="$h" "$addr" &
		SWARM_JOIN_PID[$i]=$!
	done
}

# Stops the manager.
function swarm_manage_cleanup() {
	for pid in ${SWARM_MANAGE_PID[@]}; do
		kill $pid || true
	done
}

# Clean up Swarm join processes
function swarm_join_cleanup() {
	for pid in ${SWARM_JOIN_PID[@]}; do
		kill $pid || true
	done
}

function start_docker_with_busybox() {
	# Preload busybox if not available.
	[ "$(docker_host images -q busybox)" ] || docker_host pull busybox:latest
	[ -f "$BUSYBOX_IMAGE" ] || docker_host save -o "$BUSYBOX_IMAGE" busybox:latest

	# Start the docker instances.
	local current=${#DOCKER_CONTAINERS[@]}
	start_docker "$@"
	local new=${#DOCKER_CONTAINERS[@]}

	# Load busybox on the new instances.
	for ((i=current; i < new; i++)); do
		docker -H ${HOSTS[$i]} load -i "$BUSYBOX_IMAGE"
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
			# -v /usr/local/bin -v /var/run are specific to mesos, so the slave can do a --volumes-from and use the docker cli
			docker_host run -d --name node-$i --privileged -v /usr/local/bin -v /var/run -it --net=host \
			${DOCKER_IMAGE}:${DOCKER_VERSION} \
			bash -c "\
				rm /var/run/docker.pid ; \
				rm /var/run/docker/libcontainerd/docker-containerd.pid ; \ 
				rm /var/run/docker/libcontainerd/docker-containerd.sock ; \
				hostname node-$i && \
				docker daemon -H 127.0.0.1:$port \
					-H=unix:///var/run/docker.sock \
					--storage-driver=$STORAGE_DRIVER \
					`join ' ' $@` \
		")
	done

	# Wait for the engines to be reachable.
	for ((i=current; i < (current + instances); i++)); do
		wait_until_reachable "${HOSTS[$i]}"
	done
}

# Stop all engines.
function stop_docker() {
	for id in ${DOCKER_CONTAINERS[@]}; do
		echo "Stopping $id"
		docker_host rm -f -v $id > /dev/null;
	done
}
