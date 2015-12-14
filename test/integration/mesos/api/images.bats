#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker images" {
	start_docker 1
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# With grouping, we should get 1 busybox, plus the header.
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	# Every line should contain "busybox" except for the header
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done

	# Try with --filter.
	run docker_swarm images --filter node=node-0
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]

	run docker_swarm images --filter node=node-1
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	run docker_swarm images --filter node=node-2
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	# Try images -a
	# lines are: header, busybox, <none>
	run docker_swarm images -a
	[ "${#lines[@]}" -ge 3 ]
}
