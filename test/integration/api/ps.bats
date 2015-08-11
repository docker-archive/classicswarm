#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
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

	docker_swarm run -d --name c1 busybox echo c1
	docker_swarm run -d --name c2 busybox echo c2

	run docker_swarm ps --before c1
	[ "${#lines[@]}" -eq  1 ]

	run docker_swarm ps --before c2
	[ "${#lines[@]}" -eq  2 ]

	run docker_swarm ps --before c3
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
