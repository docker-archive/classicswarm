#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker diff" {
	start_docker_with_busybox 2
	swarm_manage
	docker_swarm run -d --name test_container busybox sleep 500

	# make sure container is up
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# no changes
	run docker_swarm diff test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	# make changes on container's filesystem
	docker_swarm exec test_container touch /home/diff.txt

	# verify
	run docker_swarm diff test_container
	[ "$status" -eq 0 ]
	[[ "${output}" ==  *"diff.txt"* ]]
}
