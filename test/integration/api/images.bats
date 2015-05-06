#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker images" {
	# Start one empty host and two with busybox to ensure swarm selects the
	# right ones.
	start_docker 1
	start_docker_with_busybox 2
	swarm_manage


	# we should get 2 busyboxes, plus the header.
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 3 ]
	# Every line should contain "busybox" except for the header
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done
	
	# Try with --filter.
	run docker_swarm images --filter node=node-0
	echo $output
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm images --filter node=node-1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	run docker_swarm images --filter node=node-2
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox"* ]]
}
