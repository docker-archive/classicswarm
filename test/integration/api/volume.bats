#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker volume ls" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no volume exists
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]

	# run on node-0
	docker_swarm run -e constraint:node==node-0 -d -v=/tmp busybox true

	run docker_swarm volume ls
	echo $output
	[ "${#lines[@]}" -eq 2 ]

	docker_swarm run -e constraint:node==node-0 -d -v=/tmp busybox true

	run docker_swarm volume ls
	echo $output
	[ "${#lines[@]}" -eq 3 ]

	# create a named volume on all nodes to test --filter
	docker_swarm volume create --name=testsubstrvol

	# filter for a named volume using a name substring
	run docker_swarm volume ls --filter name=substr
	[ "$status" -eq 0 ]
	# expect 3 lines: the header and one volume per node
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[1]}" == *"testsubstrvol"* ]]
	[[ "${lines[2]}" == *"testsubstrvol"* ]]

	# filter for node-specific volumes
	# node-0 should have three volumes
	run docker_swarm volume ls --filter node=node-0
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 4 ]

	# node-1 should have one volume
	run docker_swarm volume ls --filter node=node-1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
}

@test "docker volume inspect" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d -v=/tmp -e constraint:node==node-0 busybox true

	run docker_swarm volume ls -q
	[ "${#lines[@]}" -eq 1 ]
	[[ "${output}" == *"node-0/"* ]]

	id=${output}

	run docker_swarm volume inspect $id
	[[ "${output}" == *"\"Driver\": \"local\""* ]]

	diff <(docker_swarm volume inspect $id) <(docker -H ${HOSTS[0]} volume inspect ${id#node-0/})
}

@test "docker volume create" {
	start_docker 2
	swarm_manage

	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]

	docker_swarm volume create --name=test_volume
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 3 ]

	docker_swarm run -d -v=/tmp busybox true
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 4 ]

	run docker_swarm volume create --name=node-2/test_volume2
	[ "$status" -ne 0 ]

	docker_swarm volume create --name=node-0/test_volume2
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 5 ]
}

@test "docker volume rm" {
	start_docker_with_busybox 2
	swarm_manage

	# check for failure when removing a non-existent volume
	run docker_swarm volume rm test_volume
	[ "$status" -ne 0 ]

	# run a container that exits immediately but stays around and
	# connected to the volume. Wait for it to finish.
	docker_swarm run -d --name=test_container -v=/tmp busybox true
	docker_swarm wait test_container

	run docker_swarm volume ls -q
	volume=${output}
	[ "${#lines[@]}" -eq 1 ]

	# check that removing an attached volume is an error
	run docker_swarm volume rm $volume
	[ "$status" -ne 0 ]

	docker_swarm rm test_container

	run docker_swarm volume rm $volume
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]
}

@test "docker volume ls --filter label" {
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* || "${output}" == "Docker version 1.11"* || "${output}" == "Docker version 1.12"* ]]; then
		skip
	fi
	start_docker_with_busybox 2
	swarm_manage

	# make sure no volume exists
	run docker_swarm volume ls
	[ "${#lines[@]}" -eq 1 ]

	# create a named volume on all nodes to test --filter
	docker_swarm volume create --name=testsubstrvol --label testlabel=foobar

	# filter by label
	run docker_swarm volume ls --filter label=testlabel
	[ "$status" -eq 0 ]
	# expect 3 lines: the header and one volume per node
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[1]}" == *"testsubstrvol"* ]]
	[[ "${lines[2]}" == *"testsubstrvol"* ]]

	# filter by label and value
	run docker_swarm volume ls --filter label=testlabel=foobar
	[ "$status" -eq 0 ]
	# expect 3 lines: the header and one volume per node
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[1]}" == *"testsubstrvol"* ]]
	[[ "${lines[2]}" == *"testsubstrvol"* ]]

	run docker_swarm volume ls --filter label=testlabel=notarealvalue
	[ "$status" -eq 0 ]
	# expect 1 line: just the header
	[ "${#lines[@]}" -eq 1 ]
}

@test "docker volume create with whitelist" {
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* || "${output}" == "Docker version 1.11"* || "${output}" == "Docker version 1.12"* ]]; then
		skip
	fi

	start_docker 3
	swarm_manage

	docker_swarm volume create --name=test_volume --label com.docker.swarm.whitelists=[\"node==node-1\|node-2\"]
	run docker_swarm volume ls -q
	[ "${#lines[@]}" -eq 2 ]
	[[ "${output}" != *"node-0/"* ]]
	[[ "${output}" == *"node-1/"* ]]
	[[ "${output}" == *"node-2/"* ]]
}
