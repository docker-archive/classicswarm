#!/usr/bin/env bats

load vars

@test "managing non-available node should be timed out" {

	#
	# timeout does not accept a bash function, so hard-coded path is required
	#
	run timeout 15s $GOBIN/swarm manage -H 127.0.0.1:2375 nodes://8.8.8.8:2375

	[ "$status" -ne 0 ]
	[[ ${lines[0]} =~ "Listening for HTTP" ]]
	[[ ${lines[1]} =~ "ConnectEx tcp: i/o timeout" ]]
}
