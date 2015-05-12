#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker save" {
	# Start one empty host and one with busybox to ensure swarm selects the
	# right one (and not one at random).
	start_docker 1
	start_docker_with_busybox 1
	swarm_manage

	# make sure busybox image exists
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]

	temp_file_name=$(mktemp)
	temp_file_name_o=$(mktemp)

	# save >, image->tar
	docker_swarm save busybox > $temp_file_name
	# save -o, image->tar
	docker_swarm save -o $temp_file_name_o busybox
	
	# saved image file exists, not empty and is tar file 
	[ -s $temp_file_name ]
	run file $temp_file_name
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tar archive"* ]]

	[ -s $temp_file_name_o ]
	run file $temp_file_name_o
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tar archive"* ]]

	# after ok, delete saved tar file
	rm -f $temp_file_name
	rm -f $temp_file_name_o
}

@test "docker save multi-images" {
	start_docker_with_busybox 2
	# tag busybox
	docker -H ${HOSTS[0]} tag busybox test1
	docker -H ${HOSTS[1]} tag busybox test2

	# start manage
	swarm_manage		

	# make sure image exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" == *"test1"* ]]
	[[ "${output}" == *"test2"* ]]

	local temp_file_name=$(mktemp)

	# do not support save images which are on multi machine
	run docker_swarm save -o "$temp_file_name" busybox test1 test2
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Unable to find an engine containing all images"* ]]

	# save images which are on same machine
	docker_swarm save -o "$temp_file_name" busybox test1

	# saved image file exists, not empty and is tar file
	[ -s $temp_file_name ]
	run file $temp_file_name
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tar archive"* ]]

	# Try to load our image back on an empty docker engine.
	start_docker 1
	run docker -H "${HOSTS[2]}" images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
	docker -H "${HOSTS[2]}" load -i "$temp_file_name"
	run docker -H ${HOSTS[2]} images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" == *"test1"* ]]

	rm -f $temp_file_name
}
