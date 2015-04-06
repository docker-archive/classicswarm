#!/usr/bin/env bats

load helpers

@test "test that create doesn't accept any arugments" {
	run swarm create derpderpderp
	[ "$status" -ne 0 ]
}

