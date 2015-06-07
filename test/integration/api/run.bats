#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker run" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -d --name test_container busybox sleep 100

	# verify, container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]
}

@test "docker run with resources" {
	start_docker_with_busybox 2
	swarm_manage

	# run
	docker_swarm run -d --name test_container \
			 --add-host=host-test:127.0.0.1 \
			 --cap-add=NET_ADMIN \
			 --cap-drop=MKNOD \
			 --label=com.example.version=1.0 \
			 --read-only=true \
			 --ulimit=nofile=10 \
			 --device=/dev/loop0:/dev/loop0 \
			 --ipc=host \
			 --pid=host \
			 busybox sleep 1000

	# verify, container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	run docker_swarm inspect test_container
	# label
	[[ "${output}" == *"com.example.version"* ]]
	# add-host
	[[ "${output}" == *"host-test:127.0.0.1"* ]]
	# cap-add
	[[ "${output}" == *"NET_ADMIN"* ]]
	# cap-drop
	[[ "${output}" == *"MKNOD"* ]]
	# read-only
	[[ "${output}" == *"\"ReadonlyRootfs\": true"* ]]
	# ulimit
	[[ "${output}" == *"nofile"* ]]
	# device
	[[ "${output}" == *"/dev/loop0"* ]]
	# ipc
	[[ "${output}" == *"\"IpcMode\": \"host\""* ]]
	# pid
	[[ "${output}" == *"\"PidMode\": \"host\""* ]]
}
