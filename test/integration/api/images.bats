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
	create_new_image 3
	swarm_manage

	# With grouping, we should get 1 busybox, plus the header.
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 8 ]
	# Every line should contain "busybox" except for the header
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done

	# Try with --filter by before
	run docker_swarm images --filter before=busybox
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	run docker_swarm images --filter before=busybox_1_0
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox"* ]]
	run docker_swarm images --filter before=busybox_1_1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[1]}" == *"busybox_1_0"* ]]
	[[ "${lines[2]}" == *"busybox"* ]]
	run docker_swarm images --filter before=fake_image
	[ "$status" -eq 1 ]

	# Try with --filter by since
	run docker_swarm images --filter since=busybox_2_2
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	run docker_swarm images --filter since=busybox_2_1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox_2_2"* ]]
	run docker_swarm images --filter since=busybox_2_0
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[1]}" == *"busybox_2_2"* ]]
	[[ "${lines[2]}" == *"busybox_2_1"* ]]
	run docker_swarm images --filter since=fake_image
	[ "$status" -eq 1 ]

	# Try with --filter by node
	run docker_swarm images --filter node=node-0
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm images --filter node=node-1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 5 ]
	[[ "${lines[1]}" == *"busybox_1_2"* ]]
	[[ "${lines[2]}" == *"busybox_1_1"* ]]
	[[ "${lines[3]}" == *"busybox_1_0"* ]]
	[[ "${lines[4]}" == *"busybox"* ]]

	run docker_swarm images --filter node=node-2
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 5 ]
	[[ "${lines[1]}" == *"busybox_2_2"* ]]
	[[ "${lines[2]}" == *"busybox_2_1"* ]]
	[[ "${lines[3]}" == *"busybox_2_0"* ]]
	[[ "${lines[4]}" == *"busybox"* ]]

	# Try images -a
	# lines are: header, busybox (7 lines), <none>
	run docker_swarm images -a
	[ "${#lines[@]}" -ge 9 ]

	run docker_swarm images --filter reference='busy*'
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 8 ]
	[[ "${lines[1]}" == *"busybox_2_2"* ]]
	[[ "${lines[2]}" == *"busybox_2_1"* ]]
	[[ "${lines[3]}" == *"busybox_2_0"* ]]
	[[ "${lines[4]}" == *"busybox_1_2"* ]]
	[[ "${lines[5]}" == *"busybox_1_1"* ]]
	[[ "${lines[6]}" == *"busybox_1_0"* ]]
	[[ "${lines[7]}" == *"busybox"* ]]
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
