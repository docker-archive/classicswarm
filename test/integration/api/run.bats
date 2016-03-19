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

	# error check
	run docker_swarm run -d 4e8aa3148a132f19ec560952231c4d39522043994df7d2dc239942c0f9424ebd
	[[ "${output}" == *"cannot specify 64-byte hexadecimal strings"* ]]
}

@test "docker run with image digest" {
	start_docker 2
	swarm_manage

	docker_swarm pull jimmyxian/busybox@sha256:649374debd26307573564fcf9748d39db33ef61fbf88ee84c3af10fd7e08765d

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	docker_swarm run -d --name test_container jimmyxian/busybox@sha256:649374debd26307573564fcf9748d39db33ef61fbf88ee84c3af10fd7e08765d sleep 100

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
			 --memory-swappiness=2 \
			 --group-add="root" \
			 --memory-reservation=100M \
			 --kernel-memory=100M \
			 --dns-opt="someDnsOption" \
			 --stop-signal="SIGKILL" \
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
	# memory-swappiness
	[[ "${output}" == *"\"MemorySwappiness\": 2"* ]]
	# group-add
	[[ "${output}" == *"root"* ]]
	# memory-reservation
	[[ "${output}" == *"\"MemoryReservation\": 104857600"* ]]
	# kernel-memory
	[[ "${output}" == *"\"KernelMemory\": 104857600"* ]]
	# dns-opt
	[[ "${output}" == *"someDnsOption"* ]]
	# stop-signal
	[[ "${output}" == *"\"StopSignal\": \"SIGKILL\""* ]]

	# following options are introduced in docker 1.10, skip older version
	run docker --version
	if [[ "${output}" == "Docker version 1.9"* ]]; then
		skip
	fi

	docker_swarm run -d --name test_container2 \
			 --oom-score-adj=350 \
			 --tmpfs=/tempfs:rw \
			 --device-read-iops=/dev/null:351 \
			 --device-write-iops=/dev/null:352 \
			 --device-read-bps=/dev/null:1mb \
			 --device-write-bps=/dev/null:2mb \
			 busybox sleep 1000

	run docker_swarm inspect test_container2
	# oom-score-adj
	[[ "${output}" == *"\"OomScoreAdj\": 350"* ]]
	# tmpfs
	[[ "${output}" == *"\"/tempfs\": \"rw\""* ]]
	# device-read-iops
	[[ "${output}" == *"351"* ]]
	# device-write-iops
	[[ "${output}" == *"352"* ]]
	# device-read-bps
	[[ "${output}" == *"1048576"* ]]
	# device-write-bps
	[[ "${output}" == *"2097152"* ]]
}

@test "docker run --ip" {
	# docker run --ip is introduced in docker 1.10, skip older version without --ip
	# look for --ip6 because --ip will match --ipc
	run docker run --help
	if [[ "${output}" != *"--ip6"* ]]; then
		skip
	fi

	start_docker_with_busybox 1
	swarm_manage

	docker_swarm network create -d bridge --subnet 10.0.0.0/24 testn

	docker_swarm run --name testc --net testn -d --ip 10.0.0.42 busybox sh
	run docker_swarm inspect testc
	[[ "${output}" == *"10.0.0.42"* ]]
}

@test "docker run --net-alias" {
	# docker run --net-alias is introduced in docker 1.10, skip older version without --net-alias
	run docker run --help
	if [[ "${output}" != *"--net-alias"* ]]; then
		skip
	fi

	start_docker_with_busybox 1
	swarm_manage

	docker_swarm network create -d bridge testn

	docker_swarm run --name testc --net testn -d --net-alias=testa busybox sh
	run docker_swarm inspect testc
	[[ "${output}" == *"testa"* ]]
}

@test "docker run - reschedule with image affinity" {
	start_docker_with_busybox 1
	start_docker 1

	docker -H ${HOSTS[0]} tag busybox:latest busyboxabcde:latest
	swarm_manage

	# make sure busyboxabcde exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busyboxabcde"* ]]

	# try to create container on node-1, node-1 does not have busyboxabcde and will pull it
	# but can not find busyboxabcde in dockerhub
	# then will retry with image affinity
	docker_swarm run -d --name test_container -e constraint:node==~node-1 busyboxabcde sleep 1000

	# check container running on node-0
	run docker_swarm ps
	[[ "${output}" == *"node-0/test_container"* ]]

	# check the image affinity wasn't saved
	run docker_swarm inspect test_container
	[[ "${output}" != *"image==busyboxabcde"* ]]
}

@test "docker run - reschedule with image affinity and node constraint" {
	start_docker_with_busybox 1
	start_docker 1

	docker -H ${HOSTS[0]} tag busybox:latest busyboxabcde:latest
	swarm_manage

	# make sure busyboxabcde exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busyboxabcde"* ]]

	# create container on node-1, node-1 does not have busyboxabcde and will pull it
	# but can not find busyboxabcde in dockerhub
	# because run with node constraint, will not retry with soft-image-affinity
	run docker_swarm run -d --name test_container -e constraint:node==node-1 busyboxabcde sleep 1000

	# check error message
	[[ "${output}" != *"unable to find a node that satisfies the constraint"* ]]
	[[ "${output}" == *"not found"* ]]
}

@test "docker run - constraint and soft affinities" {
	start_docker_with_busybox 1 --label group=A
	start_docker_with_busybox 1 --label group=B
	swarm_manage

	# start c0 on a node in group=A
	docker_swarm run -d --name c0 -e constraint:group==A -e affinity:container==~c0 busybox sleep 100

	# check container running on node-0
	run docker_swarm ps
	[[ "${output}" == *"node-0/c0"* ]]

	# start c2 on a node in group==B (soft affinity shouldn't matter here)
	docker_swarm run -d --name c2 -e constraint:group==B -e affinity:container==~c0 busybox sleep 100

	# check container running on a node in group=B
	run docker_swarm ps
	[[ "${output}" == *"node-1/c2"* ]]
}

@test "docker run - with not exist volume driver" {
	start_docker_with_busybox 2
	swarm_manage

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	run docker_swarm run -d --volume-driver=not_exist_volume_driver -v testvolume:/testvolume --name test_container busybox sleep 100

	# check error message
	[ "$status" -ne 0 ]
	[[ "${output,,}" == *"plugin not found"* ]]
}
