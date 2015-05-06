#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker stats" {
	TEMP_FILE=$(mktemp)
	start_docker_with_busybox 2
	swarm_manage

	# stats running container 
	run docker_swarm run -d --name test_container busybox sleep 50
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# storage the stats output in TEMP_FILE
	# it will stop automatically when manager stop
	docker_swarm stats test_container > $TEMP_FILE &

	# retry until TEMP_FILE is not empty
	retry 5 1 [ -s $TEMP_FILE ]

	# if "CPU %" in TEMP_FILE, status is 0
	run grep "CPU %" $TEMP_FILE
	[ "$status" -eq 0 ]
	run grep "MEM USAGE/LIMIT" $TEMP_FILE
	[ "$status" -eq 0 ]

	rm -f $TEMP_FILE
}
