#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker start" {
	start_docker_with_busybox 2
	swarm_manage
	# create
	docker_swarm create --name test_container busybox sleep 1000

	# make sure created container exists
	# new created container has no status
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# start
	docker_swarm start test_container

	# Verify
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]
}

@test "docker start with hostConfig" {
	start_docker_with_busybox 2
	swarm_manage
	# create
	docker_swarm create --name test_container busybox sleep 1000

	# make sure created container exists
	# new created container has no status
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# start
	curl -s -H "Content-Type: application/json" -X POST -d '{"PublishAllPorts": true}' ${SWARM_HOSTS[0]}/v1.23/containers/test_container/start

	# Verify
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# Inspect HostConfig of container, should have PublishAllPorts set to true
	run docker_swarm inspect test_container
	[[ "${output}" == *'"PublishAllPorts": true'* ]]
}