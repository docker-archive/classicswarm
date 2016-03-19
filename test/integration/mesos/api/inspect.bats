#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker inspect" {
	local version="new"
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* ]]; then
			version="old"
	fi

	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# run container
	docker_swarm run -d -m 20m -e TEST=true -h hostname.test --name test_container busybox sleep 500

	# make sure container exsists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	# inspect and verify 
	run docker_swarm inspect test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == *"NetworkSettings"* ]]
	[[ "${output}" == *"TEST=true"* ]]
	# the specific information of swarm node
	[[ "${output}" == *'"Node": {'* ]]
	[[ "${output}" == *'"Name": "node-'* ]]
	if [[ "${version}" == "old" ]]; then
		[[ "${output}" == *'"Hostname": "hostname"'* ]]
		[[ "${output}" == *'"Domainname": "test"'* ]]
	else
		[[ "${output}" == *'"Hostname": "hostname.test"'* ]]
		[[ "${output}" == *'"Domainname": ""'* ]]
	fi
}

