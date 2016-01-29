#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "rescheduling" {
	start_docker_with_busybox 2
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]},${HOSTS[1]}

	# c1 on node-0 with reschedule=on-node-failure
	run docker_swarm run -dit --name c1 -e constraint:node==~node-0 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
	[ "$status" -eq 0 ]
	# c2 on node-0 with reschedule=off
	run docker_swarm run -dit --name c2 -e constraint:node==~node-0 --label 'com.docker.swarm.reschedule-policies=["off"]' busybox sh
	[ "$status" -eq 0 ]
	# c3 on node-1
	run docker_swarm run -dit --name c3 -e constraint:node==~node-1 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
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

	# Get c1 swarm id
	swarm_id=$(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' c1)

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	# Wait for the container to be rescheduled
	retry 5 1 eval docker_swarm inspect c1

	# c1 should have been rescheduled from node-0 to node-1
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# Check swarm id didn't change for c1
	[[ "$swarm_id" == $(docker_swarm inspect -f '{{ index .Config.Labels "com.docker.swarm.id" }}' c1) ]]

	run docker_swarm inspect "$swarm_id"
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# c2 should still be on node-0 since the rescheduling policy was off.
	run docker_swarm inspect c2
	[ "$status" -eq 1 ]

	# c3 should still be on node-1 since it wasn't affected
	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]
}

@test "rescheduling with constraints" {
	start_docker_with_busybox 2
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]},${HOSTS[1]}

	# c1 on node-0 with reschedule=on-node-failure
	run docker_swarm run -dit --name c1 -e constraint:node==~node-0 -e reschedule:on-node-failure busybox sh
	[ "$status" -eq 0 ]
	# c2 on node-0 with reschedule=off
	run docker_swarm run -dit --name c2 -e constraint:node==node-0 -e reschedule:on-node-failure busybox sh
	[ "$status" -eq 0 ]
	# c3 on node-1
	run docker_swarm run -dit --name c3 -e constraint:node==node-1 -e reschedule:on-node-failure busybox sh
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
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	# Wait for the container to be rescheduled
	retry 5 1 eval docker_swarm inspect c1

	# c1 should have been rescheduled from node-0 to node-1
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# c2 should still be on node-0 since a node constraint was applied.
	run docker_swarm inspect c2
	[ "$status" -eq 1 ]

	# c3 should still be on node-1 since it wasn't affected
	run docker_swarm inspect c3
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]
}

@test "reschedule conflict" {
	start_docker_with_busybox 2
	swarm_manage

	run docker_swarm run --name c1 -dit --label 'com.docker.swarm.reschedule-policies=["false"]' busybox sh
	[ "$status" -ne 0 ]
	[[ "${output}" == *'invalid reschedule policy: false'* ]]

	run docker_swarm run --name c2 -dit -e reschedule:off --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' -e reschedule:off busybox sh
	[ "$status" -ne 0 ]
	[[ "${output}" == *'too many reschedule policies'* ]]
}

@test "rescheduling node comes back" {
	start_docker_with_busybox 2
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]},${HOSTS[1]}

	# c1 on node-0 with reschedule=on-node-failure
	run docker_swarm run -dit --name c1 -e constraint:node==~node-0 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
	[ "$status" -eq 0 ]
	# c2 on node-1
	run docker_swarm run -dit --name c2 -e constraint:node==~node-1 --label 'com.docker.swarm.reschedule-policies=["on-node-failure"]' busybox sh
	[ "$status" -eq 0 ]

	run docker_swarm ps -q
	[ "${#lines[@]}" -eq  2 ]

	# Make sure containers are running where they should.
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-0"'* ]]
	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	# Wait for the container to be rescheduled
	retry 5 1 eval docker_swarm inspect c1

	# c1 should have been rescheduled from node-0 to node-1
	run docker_swarm inspect c1
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# c2 should still be on node-1 since it wasn't affected
	run docker_swarm inspect c2
	[ "$status" -eq 0 ]
	[[ "${output}" == *'"Name": "node-1"'* ]]

	# Restart node-0
	docker_host start ${DOCKER_CONTAINERS[0]}

	sleep 5
	run docker_swarm ps
	[ "${#lines[@]}" -eq  3 ]
}
