#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker export" {
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
	echo $output

	[ "$status" -eq 0 ]
	[[ "$output" == *"tar archive"* ]]
	
	# after ok, delete exported tar file
	rm -f $temp_file_name
}
