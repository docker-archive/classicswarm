#!/usr/bin/env bats

load ../helpers

@test "cluster options" {
	# cluster option swarm.overcommit
	run swarm manage --cluster-opt swarm.overcommit=-2 nodes://192.168.56.22:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"swarm.overcommit should be larger than -1, -2.000000 is invalid"* ]]

	# cluster option swarm.createretry
	run swarm manage --cluster-opt swarm.createretry=-1 nodes://192.168.56.22:4444
	[ "$status" -ne 0 ]
	[[ "${output}" == *"swarm.createretry can not be negative, -1 is invalid"* ]]
}
