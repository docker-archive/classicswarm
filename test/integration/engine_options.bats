#!/usr/bin/env bats

load helpers

@test "engine refresh options" {
	# minimum refresh interval
	run swarm manage --engine-refresh-min-interval "0s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"minimum refresh interval should be a positive number"* ]]

	# max refresh interval 
	run swarm manage --engine-refresh-min-interval "30s" -engine-refresh-max-interval "20s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"max refresh interval cannot be less than min refresh interval"* ]]

	# engine refresh retry count
	run swarm manage --engine-refresh-retry 0 --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"invalid refresh retry count"* ]]
}
