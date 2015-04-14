#!/usr/bin/env bats
load helpers

# temp file for saving image
IMAGE_FILE=$(mktemp)

function teardown() {
    stop_manager
    stop_docker
    rm -f $IMAGE_FILE
}

@test "docker load should return success,every node should load the image" {
    # create a tar file
    docker pull busybox:latest
    docker save -o $IMAGE_FILE busybox:latest

    start_docker 1
    start_manager

    run docker_swarm -i $IMAGE_FILE
    [ "$status" -eq 0 ]
    
    run docker -H  ${HOSTS[0]} images
    [ "${#lines[@]}" -eq  2 ]
}

