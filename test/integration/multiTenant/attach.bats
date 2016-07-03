#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker attach" {
	start_docker_with_busybox 2
	swarm_manage

	# container run in background
	docker_swarm run -d -i --name test_container busybox sh -c "head -n 1; echo output"

	# inject input into the container
	attach_output=`echo input | docker_swarm attach test_container`
	# unfortunately, we cannot test `attach_output` because attach is not
	# properly returning the output (see docker/docker#12974)
	run docker_swarm logs test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == *"input"* ]]
	[[ "${output}" == *"output"* ]]
}

@test "docker attach through websocket" {
skip
	CLIENT_API_VERSION="v1.17"
	start_docker_with_busybox 2
	swarm_manage

	#create a container
	docker_swarm run -d --name test_container busybox sleep 1000

	# test attach-ws api
	# jimmyxian/centos7-wssh is an image with websocket CLI(WSSH) written in Python
	# if connected successfully, it returns two lines, "Session Open" and "Session Closed"
	# Note: with stdout=1&stdin=1&stream=1: it can be used as SSH
	URL="ws://${SWARM_HOST}/${CLIENT_API_VERSION}/containers/test_container/attach/ws?stderr=1"
	run docker_host run --rm --net=host jimmyxian/centos7-wssh wssh $URL
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Session Open"* ]]
	[[ "${output}" == *"Session Closed"* ]]
}
