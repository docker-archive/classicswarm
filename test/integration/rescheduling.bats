#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "rescheduling" {
	start_docker_with_busybox 2
	swarm_manage

	# Expect 2 nodes
	docker_swarm info | grep -q "Nodes: 2"

	# c1 on node-0 with reschedule=on-node-failure
	run docker_swarm run -dit --name c1 -e constraint:node==~node-0 --label com.docker.swarm.reschedule-policy=on-node-failure busybox sh
	[ "$status" -eq 0 ]
	# c2 on node-0 with reschedule=never
	run docker_swarm run -dit --name c2 -e constraint:node==~node-0 --label com.docker.swarm.reschedule-policy=off busybox sh
	[ "$status" -eq 0 ]
	# c3 on node-1
	run docker_swarm run -dit --name c3 -e constraint:node==~node-1 --label com.docker.swarm.reschedule-policy=on-node-failure busybox sh
	[ "$status" -eq 0 ]

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  3 ]

	# Make sure containers are running where they should.
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]
	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]
	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	#retry 10 1 eval "docker_swarm info | grep -q 'Nodes: 1'"

	sleep 5
	docker_swarm ps

	# c1 should have been rescheduled from node-0 to node-1
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# c2 should still be on node-0 since the rescheduling policy was off.
	run docker_swarm inspect c2
	[ "$status" -eq 1 ]

	# c3 should still be on node-1 since it wasn't affected
	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}
