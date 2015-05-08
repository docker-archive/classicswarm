#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker events" {
	TEMP_FILE=$(mktemp)
	start_docker_with_busybox 2
	swarm_manage

	# start events, report real time events to TEMP_FILE
	# it will stop automatically when manager stop
	docker_swarm events > $TEMP_FILE &

	# events: create container on node-0
	docker_swarm create --name test_container -e constraint:node==node-0 busybox sleep 100 
	
	# events: start container
	docker_swarm start test_container

	# verify
	run cat $TEMP_FILE
	[ "$status" -eq 0 ]
	[[ "${output}" == *"node:node-0"* ]]
	[[ "${output}" == *"create"* ]]
	[[ "${output}" == *"start"* ]]
	
	# after ok, remove the $TEMP_FILE
	rm -f $TEMP_FILE
}
