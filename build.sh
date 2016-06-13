#!/bin/bash

GITCOMMIT=${1:-''}
DATE=${2:-''}
GITREMOTE=${3:-''}
IMAGE_NAME=${4:-radiant/swarm}
BUILD_ID=${5:-0}
BUILD_NUMBER=${6:-0}
GIT_TAG=${7:-''}
BUILDER_IMAGE=radiant/swarm_build

PROJECT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# build the binaries
docker build -t $BUILDER_IMAGE -f $PROJECT_DIR/Dockerfile.radiant.build $PROJECT_DIR

# copy out the binaries
docker run --rm -v $PROJECT_DIR:/tmp:rw $BUILDER_IMAGE cp /go/bin/swarm /tmp

# build the image
docker build \
--build-arg git_commit_id=${GITCOMMIT} \
--build-arg build_id=${BUILD_ID} \
--build-arg build_number=${BUILD_NUMBER} \
--build-arg build_date=${DATE} \
--build-arg git_tag=${GIT_TAG} \
--build-arg git_remote_url=${GITREMOTE} \
-f $PROJECT_DIR/Dockerfile.radiant.scratch \
-t $IMAGE_NAME --no-cache $PROJECT_DIR

# remove the binary
rm $PROJECT_DIR/swarm
