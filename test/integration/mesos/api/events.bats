#!/usr/bin/env bats

load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker events" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	# start events, report real time events to $log_file
	local log_file=$(mktemp)
	docker_swarm events > "$log_file" &
	local events_pid="$!"

	# This should emit 3 events: create, start, die.
	docker_swarm run -d -m 20m --name test_container -e constraint:node==node-0 busybox true
	
	# events might take a little big to show up, wait until we get the last one.
	retry 5 0.5 grep -q "die" "$log_file"

	# clean up `docker events`
	kill "$events_pid"

	# verify
	run cat "$log_file"
	[ "$status" -eq 0 ]
	[[ "${output}" == *"node:node-0"* ]]
	[[ "${output}" == *"create"* ]]
	[[ "${output}" == *"start"* ]]
	[[ "${output}" == *"die"* ]]
	
	# after ok, remove the log file
	rm -f "$log_file"
}
