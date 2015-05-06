@test "docker save" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	temp_file_name=$(mktemp)
	temp_file_name_o=$(mktemp)
	# make sure busybox image exists
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]

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
