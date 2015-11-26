#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker import" {
	start_docker_with_busybox 2
	swarm_manage
	# run a container to export
	docker_swarm run -d --name test_container busybox sleep 500

	temp_file_name=$(mktemp)
	# make sure container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	# export, container->tar
	docker_swarm export test_container > $temp_file_name

	# verify: exported file exists, not empty and is tar file
	[ -s $temp_file_name ]
	run file $temp_file_name
	[ "$status" -eq 0 ]
	[[ "$output" == *"tar archive"* ]]

	# import
	docker_swarm import - testbusybox < $temp_file_name

	# verify on the nodes
	for host in ${HOSTS[@]}; do
		run docker -H $host images
		[ "$status" -eq 0 ]
		[[ "${output}" == *"testbusybox"* ]]
	done
	
	# after ok, delete exported tar file
	rm -f $temp_file_name
}

@test "docker import - check error code" {
	start_docker 2
	swarm_manage

	temp_file=$(mktemp)
	echo abc > $temp_file

	run docker_swarm import - < $temp_file
	[ "$status" -eq 1 ]

	rm -f $temp_file
}
