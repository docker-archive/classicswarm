#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "scheduler avoids failing node" {
	# Docker issue #14203 in runC causing this test to fail.
	# Issue fixed after Docker 1.10
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* ]]; then
		skip
	fi

	# Start 1 engine and register it in the file.
	start_docker 2
	# Start swarm and check it can reach the node
	# refresh interval is 20s. 20 retries before marking it as unhealthy
	swarm_manage --engine-refresh-min-interval "20s" --engine-refresh-max-interval "20s" --engine-failure-retry 20 "${HOSTS[0]},${HOSTS[1]}"

	eval "docker_swarm info | grep -q -i 'Nodes: 2'"

	# Use memory on node-0
	docker_swarm run -e constraint:node==node-0 -m 50m busybox sh

	# Stop the node-1
	docker_host stop ${DOCKER_CONTAINERS[1]}

	# Try to schedule a container. It'd first select node-1 and fail
	run docker_swarm run -m 10m busybox sh
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Cannot connect to the docker engine endpoint"* ]]

	# Try to run it again. It'd select node-0 and succeed
	run docker_swarm run -m 10m busybox sh
	[ "$status" -eq 0 ]
}

@test "refresh loop detects failure" {
	# Docker issue #14203 in runC causing this test to fail.
	# Issue fixed after Docker 1.10
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* ]]; then
		skip
	fi

	# Start 1 engine and register it in the file.
	start_docker 2
	# Start swarm and check it can reach the node
	# refresh interval is 1s. 20 retries before marking it as unhealthy
	swarm_manage --engine-refresh-min-interval "1s" --engine-refresh-max-interval "1s" --engine-failure-retry 20 "${HOSTS[0]},${HOSTS[1]}"

	eval "docker_swarm info | grep -q -i 'Nodes: 2'"

	# Use memory on node-0
	docker_swarm run -e constraint:node==node-0 -m 50m busybox sh

	# Stop the node-1
	docker_host stop ${DOCKER_CONTAINERS[1]}

	# Sleep to let refresh loop detect node-1 failure
	sleep 3

	# Try to schedule a container. It'd select node-0 and succeed
	run docker_swarm run -m 10m busybox sh
	[ "$status" -eq 0 ]
}

@test "scheduler retry" {
	# Docker issue #14203 in runC causing this test to fail.
	# Issue fixed after Docker 1.10
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* || "${output}" == "Docker version 1.10"* ]]; then
		skip
	fi

	# Start 1 engine and register it in the file.
	start_docker 2
	# Start swarm and check it can reach the node
	# refresh interval is 20s. 20 retries before marking it as unhealthy
	swarm_manage --engine-refresh-min-interval "20s" --engine-refresh-max-interval "20s" --engine-failure-retry 20 -cluster-opt swarm.createretry=1 "${HOSTS[0]},${HOSTS[1]}"

	eval "docker_swarm info | grep -q -i 'Nodes: 2'"

	# Use memory on node-0
	docker_swarm run -e constraint:node==node-0 -m 50m busybox sh

	# Stop the node-1
	docker_host stop ${DOCKER_CONTAINERS[1]}

	# Try to run a container. It'd try node-1, upon failure automatically retry on node-0
	run docker_swarm run -m 10m busybox sh
	[ "$status" -eq 0 ]
}
