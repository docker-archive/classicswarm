#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

# Ensure that the client and server are running the same version.
@test "api version" {
	start_docker 1
	run docker -H "${HOSTS[0]}" version
	[ "$status" -eq 0 ]

	# First line should contain the client version.
	[[ "${lines[0]}" == "Client version: "* ]]
	local cli_version=`echo "${lines[0]}" | cut -d':' -f2`
	[[ "${output}" == *"Server version:$cli_version"* ]]

	# Second line should be client API version.
	[[ "${lines[1]}" == "Client API version: "* ]]
	local cli_api_version=`echo "${lines[1]}" | cut -d':' -f2`
	[[ "${output}" == *"Server API version:$cli_api_version"* ]]
}
