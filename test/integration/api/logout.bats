#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

# FIXME
@test "docker logout" {
	skip
}
