#!/usr/bin/env bats

load helpers

# Discovery parameter for Swarm
DISCOVERY="consul://127.0.0.1:5555/test"

@test "swarm join" {
	# --advertise
	run swarm join --heartbeat=1s --ttl=10s --delay=1s --advertise="" "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"missing mandatory --advertise flag"* ]]

	run swarm join --heartbeat=1s --ttl=10s --delay=1s --advertise=127.0.0.1ac:sh25 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--advertise should be of the form ip:port or hostname:port"* ]]

	run swarm join --heartbeat=1s --ttl=10s --delay=1s --advertise=127.0.0.1:0 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--advertise should be of the form ip:port or hostname:port"* ]]

	run swarm join --heartbeat=1s --ttl=10s --delay=1s --advertise=127.0.0.1:65536 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--advertise should be of the form ip:port or hostname:port"* ]]

	# --delay
	run swarm join --heartbeat=1s --ttl=10s --delay=asdf --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]

	run swarm join --heartbeat=1s --ttl=10s --delay=-30s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--delay should not be a negative number"* ]]

	# --heartbeat
	run swarm join --heartbeat=asdf --ttl=10s --delay=1s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]

	run swarm join --heartbeat=-10s --ttl=10s --delay=1s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--heartbeat should be at least one second"* ]]

	# --ttl
	run swarm join --heartbeat=1s --ttl=asdf --delay=1s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]

	run swarm join --heartbeat=1s --ttl=-10s --delay=1s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--ttl must be strictly superior to the heartbeat value"* ]]

	run swarm join --heartbeat=2s --ttl=1s --delay=1s --advertise=127.0.0.1:2376 "$DISCOVERY"
	[ "$status" -ne 0 ]
	[[ "${output}" == *"--ttl must be strictly superior to the heartbeat value"* ]]
}
