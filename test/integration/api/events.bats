#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker events" {
	start_docker_with_busybox 2
	swarm_manage

	# start events, report real time events to $log_file
	local log_file=$(mktemp)
	docker_swarm events > "$log_file" &
	local events_pid="$!"

	# This should emit 3 events: create, start, die.
	docker_swarm run -d --name test_container -e constraint:node==node-0 busybox true

	# events might take a little big to show up, wait until we get the last one.
	retry 5 0.5 grep -q "die" "$log_file"

	# clean up `docker events`
	kill "$events_pid"

	# verify size
	[[ $(wc -l < ${log_file}) -ge 3 ]]

	# verify content
	run cat "$log_file"
	[ "$status" -eq 0 ]
	[[ "${output}" == *"node-0"* ]]
	[[ "${output}" == *"create"* ]]
	[[ "${output}" == *"start"* ]]
	[[ "${output}" == *"die"* ]]
	
	# after ok, remove the log file
	rm -f "$log_file"
}


@test "docker events until" {
	# produce less output because we timed out
	start_docker_with_busybox 2
	swarm_manage

	# start events, report real time events to $log_file
	local log_file=$(mktemp)
	ONE_SECOND_IN_THE_PAST=$(($(date +%s) - 1))
	docker_swarm events --until ${ONE_SECOND_IN_THE_PAST} > "$log_file"

	# This should emit 3 events: create, start, die.
	docker_swarm run --name test_container -e constraint:node==node-0 busybox true

	# do not need to kill events, it's already dead

	# verify size
	[[ $(wc -l < ${log_file}) == 0 ]]
	# no content, so nothing else to verify

	# after ok, remove the log file
	rm -f "$log_file"
}
