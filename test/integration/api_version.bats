#!/usr/bin/env bats

load helpers

function teardown() {
	stop_docker
}

# Ensure that the client and server are running the same version.
#
# If this test is failing, it means that your test environment is misconfigured
# and your host CLI version differs from DOCKER_VERSION.
@test "api version" {
	start_docker 1

	# Get version output
	out=$(docker -H "${HOSTS[0]}" version)	

	# Check client version
	run bash -c "echo '$out' | egrep -i '^\s*Version'"
	[ "$status" -eq 0 ]

	[[ $(echo "${lines[0]}" | cut -d':' -f2 | awk -F' ' '{print $1}')  == $(echo "${lines[1]}" | cut -d':' -f2 | awk -F' ' '{print $1}') ]]

	# Check API version
	run bash -c "echo '$out' | egrep -i '^\s*API version:'"
	[ "$status" -eq 0 ]

	[[ $(echo "${lines[0]}" | cut -d':' -f2 | awk -F' ' '{print $1}')  == $(echo "${lines[1]}" | cut -d':' -f2 | awk -F' ' '{print $1}') ]]

}
