#!/usr/bin/env bats

load helpers

# Discovery parameter for Swarm
DISCOVERY="consul://127.0.0.1:5555/test"

@test "swarm list" {
	# --timeout
	run swarm list --timeout "-10s" "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--timeout should be a positive number"* ]]

	# --timeout
	run swarm list --timeout "0s" "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--timeout should be a positive number"* ]]
}
