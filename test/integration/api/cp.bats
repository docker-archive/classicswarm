#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker cp" {
	start_docker_with_busybox 2
	swarm_manage

	test_file="/bin/busybox"
	# create a temporary destination directory
	temp_dest=`mktemp -d`

	# create the container
	docker_swarm run -d --name test_container busybox sleep 500

	# make sure container is up
	# FIXME(#748): Retry required because of race condition.
	retry 5 0.5 eval "[ $(docker_swarm inspect -f '{{ .State.Running }}' test_container) == 'true' ]"

	# grab the checksum of the test file inside the container.
	run docker_swarm exec test_container md5sum $test_file
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 1 ]

	# get the checksum number
	container_checksum=$(echo ${lines[0]} | awk '{print $1}')

	# host file
	host_file=$temp_dest/$(basename $test_file)
	[ ! -f $host_file ]

	# copy the test file from the container to the host.
	docker_swarm cp test_container:$test_file $temp_dest
	[ -f $host_file ]

	# compute the checksum of the copied file.
	run md5sum $host_file
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 1 ]
	host_checksum=$(echo ${lines[0]} | awk '{print $1}')

	# Verify that they match.
	[ "${container_checksum}" == "${host_checksum}" ]
	# after ok, remove temp directory and file 
	rm -rf "$temp_dest"
}
