#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker update" {
	# docker update is introduced in docker 1.10, skip older version without update command
	run docker help update
	if [[ "${output}" != *"Usage:	docker update"* ]]; then
		skip
	fi

	start_docker_with_busybox 1
	swarm_manage
	docker_swarm run -d --name test_container \
				-m=10M \
				--cpu-period=50000 \
				--cpu-quota=25000 \
				--blkio-weight=300 \
			busybox sleep 100

	run docker_swarm inspect test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == *"\"Memory\": 10485760"* ]]
	[[ "${output}" == *"\"CpuPeriod\": 50000"* ]]
	[[ "${output}" == *"\"CpuQuota\": 25000"* ]]
	[[ "${output}" == *"\"BlkioWeight\": 300"* ]]

	# validate docker info reflects the resource change
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 10 MiB"* ]]

	run docker_swarm update \
				-m=100M \
				--cpu-period=50000 \
				--cpu-quota=5000 \
				--blkio-weight=600 \
			test_container
	[ "$status" -eq 0 ]

	run docker_swarm inspect test_container
	[ "$status" -eq 0 ]
	[[ "${output}" == *"\"Memory\": 104857600"* ]]
	[[ "${output}" == *"\"CpuQuota\": 5000"* ]]
	[[ "${output}" == *"\"BlkioWeight\": 600"* ]]

	# validate docker info reflects the resource change
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Reserved Memory: 100 MiB"* ]]
}
