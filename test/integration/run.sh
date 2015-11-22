#!/usr/bin/env bash
set -e

# try to use gnu version of readlink on non-gnu systems (e.g. bsd, osx)
# on osx, install with 'brew install coreutils'
READLINK_LOCATION=$(which greadlink readlink | head -n 1)
THIS_SCRIPT=$(${READLINK_LOCATION} -f "$BASH_SOURCE")
cd "$(dirname "${THIS_SCRIPT}")"

# Root directory of Swarm.
SWARM_ROOT=$(cd ../..; pwd -P)

# Image containing the integration tests environment.
INTEGRATION_IMAGE=${INTEGRATION_IMAGE:-dockerswarm/swarm-test-env}

# Make sure we upgrade the integration environment.
docker pull $INTEGRATION_IMAGE

# Start the integration tests in a Docker container.
ID=$(docker run -d -t --privileged \
	-v /sys/fs/cgroup:/sys/fs/cgroup:ro `# this is specific to mesos` \
	-v ${SWARM_ROOT}:/go/src/github.com/docker/swarm \
	-e "DOCKER_IMAGE=$DOCKER_IMAGE" \
	-e "DOCKER_VERSION=$DOCKER_VERSION" \
	-e "STORAGE_DRIVER=$STORAGE_DRIVER" \
	${INTEGRATION_IMAGE} \
	./test_runner.sh "$@")

# Clean it up when we exit.
trap "docker rm -f -v $ID > /dev/null" EXIT

docker logs -f $ID
