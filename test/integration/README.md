# Swarm Integration Tests

Integration tests provide end-to-end testing of Swarm.

While unit tests verify the code is working as expected by relying on mocks and
artificially created fixtures, integration tests actually use real docker
engines and communicate with Swarm through the CLI.

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

The tests expect the *swarm* binary to be built and located at the root
directory of the repo:
```
$ godep go build
```

In order to run all integration tests, pass *bats* the test path:
```
$ bats test/integration
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
	# start_docker will start a given number of engines:
	start_docker 2

	# The address of those engines is available through $HOSTS:
	run docker -H ${HOSTS[0]} info
	# "run" automatically populates $status, $output and $lines.
	# Please refer to bats documentation to find out more.
	[ "$status" -eq 0 ]

	# You can start multiple docker engines in the same test:
	start_docker 1
	# Their address will be *appended* to $HOSTS. Therefore, since this is the 3rd
	# engine started, its index will be 2:
	run docker -H ${HOSTS[2]} info  

	# Additional parameters can be passed to the docker daemon:
	start_docker 1 --label foo=bar

	# swarm_manage will start the swarm manager, preconfigured with all engines
	# previously started by start_engine:
	swarm_manage

	# You can talk with said manager by using the docker_swarm helper:
	run docker_swarm info
	[ "$status" -eq 0 ]
	[ "${lines[1]}"="Nodes: 4" ]
}

# teardown is called at the end of every test.
function teardown() {
	# This will stop the swarm manager:
	swarm_manage_cleanup

	# This will stop and remove all engines launched by the test:
	stop_docker
}
```
