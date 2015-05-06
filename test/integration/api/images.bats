#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker images" {
	start_docker 3
	swarm_manage

	# no image exist
	run docker_swarm images -q 
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
	# make sure every node has no image
	for((i=0; i<3; i++)); do
		run docker_swarm images --filter node=node-$i -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done

	# pull image
	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# show all node images, including reduplicated
	run docker_swarm images
	[ "$status" -eq 0 ]
	# check pull busybox, if download sucessfully, the busybox have one tag(lastest) at least
	# if there are 3 nodes, the output lines of "docker images" are greater or equal 4(1 header + 3 busybox:latest)
	# so use -ge here, the following(pull/tag) is the same reason
	[ "${#lines[@]}" -ge 4 ]
	# Every line should contain "busybox" exclude the first head line 
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done
	
	# verify
	for((i=0; i<3; i++)); do
		run docker_swarm images --filter node=node-$i
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -ge 2 ]
		[[ "${lines[1]}" == *"busybox"* ]]
	done
}
