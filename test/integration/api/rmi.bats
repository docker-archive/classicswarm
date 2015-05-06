@test "docker rmi" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure image exists
	# swarm check image
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]
	# node check image
	for host in ${HOSTS[@]}; do
		run docker -H $host images
		[ "$status" -eq 0 ]
		[[ "${output}" == *"busybox"* ]]
	done

	# this test presupposition: do not run image
	run docker_swarm rmi busybox
	[ "$status" -eq 0 ]

	# swarm verify
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
	# node verify
	for host in ${HOSTS[@]}; do
		run docker -H $host images -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done
}
