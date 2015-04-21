# Mesos Swarm Integration Tests

Integration tests provide integrated features of Mesos Swarm tests.

While unit tests verify the code is working as expected by relying on mocks and
artificially created fixtures, integration tests actually use real docker
engines and communicate and schedule resources with Swarm through Mesos.

Note that integration tests do **not** replace unit tests.

As a rule of thumb, code should be tested thoroughly with unit tests.
Integration tests on the other hand are meant to test a specific feature end
to end.

Integration tests are written in *bash* using the
[bats](https://github.com/sstephenson/bats) framework.

## Running integration tests

Start by [installing]
(https://github.com/sstephenson/bats#installing-bats-from-source) *bats* on
your system.

In order to run all integration tests, pass *bats* the test path:
```
$ bats test/integration_mesos
```

## Writing integration tests

[helper functions]
(https://github.com/docker/swarm/blob/master/test/integration/helpers.bash)
are provided in order to facilitate writing tests.

```sh
#!/usr/bin/env bats

# This will load the helpers.
load helpers

@test "this is a simple test" {
	# Unlike swarm integration tests, Mesos Swarm will start docker on Mesos nodes.
	# swarm_mesos_manage will start the swarm manager preconfigured to run on a 
	# Mesos cluster:
	swarm_manage

	# You can talk with said manager by using the docker_swarm helper:
	run docker_swarm info
	[ "$status" -eq 0 ]
}

# teardown is called at the end of every test.
function teardown() {
	# This will stop the swarm manager:
	swarm_mesos_manage_cleanup
}
```
