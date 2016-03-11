#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker ps - host down" {
	start_docker_with_busybox 2
	swarm_manage --engine-refresh-min-interval=1s --engine-refresh-max-interval=1s --engine-failure-retry=1 ${HOSTS[0]},${HOSTS[1]}

	docker_swarm run -d -e constraint:node==node-0 busybox sleep 42
	docker_swarm run -d -e constraint:node==node-1 busybox sleep 42

	run docker_swarm ps
	[ "${#lines[@]}" -eq  3 ]

	# Stop node-0
	docker_host stop ${DOCKER_CONTAINERS[0]}

	# Wait for Swarm to detect the node failure.
	retry 5 1 eval "docker_swarm info | grep -q 'Unhealthy'"

	run docker_swarm ps
	# container with host down shouldn't be displyed since they are not `running`
	[ "${#lines[@]}" -eq  2 ]

	run docker_swarm ps -a
	[ "${#lines[@]}" -eq  3 ]
}

@test "docker ps -n" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d busybox sleep 42
	docker_swarm run -d busybox false
	run docker_swarm ps -n 3
	# Non-running containers should be included in ps -n
	[ "${#lines[@]}" -eq  3 ]

	docker_swarm run -d busybox true
	run docker_swarm ps -n 3
	[ "${#lines[@]}" -eq  4 ]

	docker_swarm run -d busybox true
	run docker_swarm ps -n 3
	[ "${#lines[@]}" -eq  4 ]
}

@test "docker ps -l" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d busybox sleep 42
	sleep 1 #sleep so the 2 containers don't start at the same second
	docker_swarm run -d busybox true
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq  2 ]
	# Last container should be "true", even though it's stopped.
	[[ "${lines[1]}" == *"true"* ]]

	sleep 1 #sleep so the container doesn't start at the same second as 'busybox true'
	run docker_swarm run -d busybox false
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"false"* ]]
}

@test "docker ps --before" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run -d --name container1 busybox echo container1
	sleep 1 #sleep so the 2 containers don't start at the same second
	docker_swarm run -d --name container2 busybox echo container2

	run eval "docker_swarm ps --before container1 2>/dev/null"
	[ "${#lines[@]}" -eq  1 ]

	run eval "docker_swarm ps --before container2 2>/dev/null"
	[ "${#lines[@]}" -eq  2 ]

	run docker_swarm ps --before container3
	[ "$status" -eq 1 ]
}

@test "docker ps --filter" {
	start_docker_with_busybox 2
	swarm_manage

	# Running
	firstID=$(docker_swarm run -d --name name1 --label "match=me" --label "second=tag" busybox sleep 10000)
	# Exited - successful
	secondID=$(docker_swarm run -d --name name2 --label "match=me too" busybox true)
	docker_swarm wait "$secondID"
	# Exited - error
	thirdID=$(docker_swarm run -d --name name3 --label "nomatch=me" busybox false)
	docker_swarm wait "$thirdID"
	# Exited - error

	# status
	run docker_swarm ps -q --no-trunc --filter=status=exited
	echo $output
	[ "${#lines[@]}" -eq  2 ]
	[[ "$output" != *"$firstID"* ]]
	[[ "$output" == *"$secondID"* ]]
	[[ "$output" == *"$thirdID"* ]]
	run docker_swarm ps -q -a --no-trunc --filter=status=running
	[[ "$output" == "$firstID" ]]

	# id
	run docker_swarm ps -a -q --no-trunc --filter=id="$secondID"
	[[ "$output" == "$secondID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=id="bogusID"
	[ "${#lines[@]}" -eq  0 ]

	# name
	run docker_swarm ps -a -q --no-trunc --filter=name=name3
	[[ "$output" == "$thirdID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=name=badname
	[ "${#lines[@]}" -eq  0 ]

	# exit code
	run docker_swarm ps -a -q --no-trunc --filter=exited=0
	[[ "$output" == "$secondID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=exited=1
	[[ "$output" == "$thirdID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=exited=99
	[ "${#lines[@]}" -eq  0 ]

	# labels
	run docker_swarm ps -a -q --no-trunc --filter=label=match=me
	[[ "$output" == "$firstID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=label=match=me --filter=label=second=tag
	[[ "$output" == "$firstID" ]]
	run docker_swarm ps -a -q --no-trunc --filter=label=match=me --filter=label=second=tag-no
	[ "${#lines[@]}" -eq  0 ]
	run docker_swarm ps -a -q --no-trunc --filter=label=match
	[ "${#lines[@]}" -eq  2 ]
	[[ "$output" == *"$firstID"* ]]
	[[ "$output" == *"$secondID"* ]]
	[[ "$output" != *"$thirdID"* ]]
}

@test "docker ps --filter node" {
	start_docker_with_busybox 2
	swarm_manage

	docker_swarm run --name c1 -e constraint:node==node-0 -d busybox:latest sleep 100
	docker_swarm run --name c2 -e constraint:node==node-1 -d busybox:latest sleep 100

	run docker_swarm ps --filter node=node-0
	[ "$status" -eq 0 ]
	[[ "${output}" == *"node-0/c1"* ]]
	[[ "${output}" != *"node-1/c2"* ]]
}
