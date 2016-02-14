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

	# With grouping, we should get 1 busybox, plus the header.
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	# Every line should contain "busybox" except for the header
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done

	# Try with --filter.
	run docker_swarm images --filter node=node-0
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

	# Try images -a
	# lines are: header, busybox, <none>
	run docker_swarm images -a
	[ "${#lines[@]}" -ge 3 ]
}

@test "docker images -f label" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm build -t image-with-labels $TESTDATA/imagelabel

	run docker_swarm images \
		--filter label=com.docker.swarm.test.integration.images=labeltest
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"image-with-labels"* ]]
}

@test "docker images imagetag" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm build -t testimage:latest $TESTDATA/imagelabel

	run docker_swarm images testimage
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"testimage"* ]]
}

@test "docker images - after commit on engine side" {
	start_docker_with_busybox 2
	swarm_manage

	docker -H ${HOSTS[0]} run -d --name test_container busybox sleep 500
	docker -H ${HOSTS[0]} commit test_container testimage

	retry 5 1 eval "docker_swarm images | grep 'testimage'"
}
