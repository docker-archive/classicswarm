#!/usr/bin/env bash
set -e

cd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

RELEASE_IMAGE="dockerswarm/dind"
RELEASE_VERSIONS="1.6.0 1.5.0 1.4.0"

# Master.
echo "+++ Testing against docker master"
DOCKER_IMAGE="dockerswarm/dind-master" DOCKER_VERSION="latest" bash "../integration/run.sh"

# Releases.
for version in $RELEASE_VERSIONS; do
	export DOCKER_IMAGE="$RELEASE_IMAGE"
	export DOCKER_VERSION="$version"
	
	echo "+++ Testing with $version"
	bash "../integration/run.sh"
done
