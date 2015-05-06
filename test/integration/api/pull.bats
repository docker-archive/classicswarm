@test "docker pull" {
	start_docker 3
	swarm_manage

	# make sure no image exists
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# swarm verify
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 4 ]
	# every line should contain "busybox" exclude the first head line 
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done

	# node verify
	for host in ${HOSTS[@]}; do
		run docker -H $host images
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -ge 2 ]
		[[ "${lines[1]}" == *"busybox"* ]]
	done
}
