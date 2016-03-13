#!/usr/bin/env bats

load ../helpers

@test "engine refresh options" {
	# minimum refresh interval
	run swarm manage --engine-refresh-min-interval "0s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"min refresh interval should be a positive number"* ]]

	run swarm manage --engine-refresh-min-interval "-10s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"min refresh interval should be a positive number"* ]]

	# maximum refresh interval, minimum refresh interval is 30s as default
	run swarm manage --engine-refresh-max-interval "0s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"max refresh interval cannot be less than min refresh interval"* ]]

	run swarm manage --engine-refresh-max-interval "-30s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"max refresh interval cannot be less than min refresh interval"* ]]

	# max refresh interval is larger than min refresh interval
	run swarm manage --engine-refresh-min-interval "30s" -engine-refresh-max-interval "20s" --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"max refresh interval cannot be less than min refresh interval"* ]]

	# engine refresh retry count
	run swarm manage --engine-failure-retry 0 --advertise 127.0.0.1:$SWARM_BASE_PORT 192.168.56.202:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"invalid failure retry count"* ]]
}
