#!/usr/bin/env bats

load vars

@test "version string should contain a proper number with git commit" {
	run swarm -v

	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ version\ [0-9]+\.[0-9]+\.[0-9]+\ \([0-9a-f]{7}\)$ ]]
}
