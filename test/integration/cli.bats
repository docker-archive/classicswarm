#!/usr/bin/env bats

load helpers

@test "swarm create doesn't accept any arguments" {
	run swarm create derpderpderp
	[ "$status" -ne 0 ]
}

@test "swarm version" {
	run swarm -v

	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ version\ [0-9]+\.[0-9]+\.[0-9]+ ]]
}
