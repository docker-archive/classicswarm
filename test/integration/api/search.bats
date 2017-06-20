#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker search" {
	start_docker 2
	swarm_manage

	local version="new"
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* || "${output}" == "Docker version 1.11"* || "${output}" == "Docker version 1.12"* || "${output}" == "Docker version 1.13"* || "${output}" == "Docker version 17.03"* ]]; then
			version="old"
	fi

	# search image (not exist), the name of images only [a-z0-9-_.] are allowed
	run docker_swarm search does_not_exist
	[ "$status" -eq 0 ]

	# for newer versions, a warning is printed out as well, which takes up the first line of the output
	if [[ "${version}" == "old" ]]; then
		[ "${#lines[@]}" -eq 1 ]
		[[ "${lines[0]}" == *"DESCRIPTION"* ]]
	else
		[ "${#lines[@]}" -eq 2 ]
		[[ "${lines[0]}" == *"Warning: Empty registry endpoint from daemon"* ]]
		[[ "${lines[1]}" == *"DESCRIPTION"* ]]
	fi

	# search busybox (image exist)
	run docker_swarm search busybox
	[ "$status" -eq 0 ]

	# search existed image, output: latest + header at least
	if [[ "${version}" == "old" ]]; then
		[ "${#lines[@]}" -ge 2 ]
		# Every line should contain "busybox" exclude the first head line
		for((i=1; i<${#lines[@]}; i++)); do
			[[ "${lines[i]}" == *"busybox"* ]]
		done
	else
		[ "${#lines[@]}" -ge 3 ]
		# Every line should contain "busybox" exclude the first warning and second head line
		for((i=2; i<${#lines[@]}; i++)); do
			[[ "${lines[i]}" == *"busybox"* ]]
		done
	fi
}
