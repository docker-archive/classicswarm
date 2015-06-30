#!/usr/bin/env bash
set -e

cd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

# Load the helpers.
. helpers.bash

function execute() {
	>&2 echo "++ $@"
	eval "$@"
}

# Tests to run. Defaults to all.
TESTS=${@:-. discovery api } #mesos

# Generate a temporary binary for the tests.
export SWARM_BINARY=`mktemp`

# Build Swarm.
execute time go build -o "$SWARM_BINARY" ../..

# Start the docker engine.
execute docker --daemon --log-level=panic \
	--storage-driver="$STORAGE_DRIVER" --exec-driver="$EXEC_DRIVER" &
DOCKER_PID=$!

# Wait for it to become reachable.
tries=10
until docker version &> /dev/null; do
	(( tries-- ))
	if [ $tries -le 0 ]; then
		echo >&2 "error: daemon failed to start"
		exit 1
	fi
	sleep 1
done

# Pre-fetch the test image.
execute time docker pull ${DOCKER_IMAGE}:${DOCKER_VERSION} > /dev/null

# Run the tests using the same client provided by the test image.
id=`execute docker create ${DOCKER_IMAGE}:${DOCKER_VERSION}`
tmp=`mktemp -d`
execute docker cp "${id}:/usr/local/bin/docker" "$tmp"
execute docker rm -f "$id" > /dev/null
export DOCKER_BINARY="${tmp}/docker"

# Run the tests.
execute time bats -p $TESTS
