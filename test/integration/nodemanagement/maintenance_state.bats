#!/usr/bin/env bats

# This will load the helpers.
load ../helpers

# teardown is called at the end of every test.
function teardown() {
    # This will stop the swarm manager:
    swarm_manage_cleanup

    # This will stop and remove all engines launched by the test:
    stop_docker
}

@test "Maintenance State: Non existing container" {
    # start_docker will start a given number of engines:
    start_docker 1

    # The address of those engines is available through $HOSTS:
    run docker -H ${HOSTS[0]} info
    # "run" automatically populates $status, $output and $lines.
    # Please refer to bats documentation to find out more.

    [ "$status" -eq 0 ]

    # swarm_manage will start the swarm manager, preconfigured with all engines
    # previously started by start_engine:
    swarm_manage

    # You can talk with said manager by using the docker_swarm helper:
    run docker_swarm info
    [ "$status" -eq 0 ]
    [ "${lines[1]}"="Nodes: 1" ]

    run curl localhost:${SWARM_BASE_PORT}/engines/foo/maintenance

    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"No such container: foo"* ]]
}

@test "Maintenance State: Getting maintenance state" {
    # start_docker will start a given number of engines:
    start_docker 3

    # The address of those engines is available through $HOSTS:
    run docker -H ${HOSTS[0]} info
    # "run" automatically populates $status, $output and $lines.
    # Please refer to bats documentation to find out more.

    [ "$status" -eq 0 ]

    # swarm_manage will start the swarm manager, preconfigured with all engines
    # previously started by start_engine:
    swarm_manage

    # You can talk with said manager by using the docker_swarm helper:
    run docker_swarm info
    [ "$status" -eq 0 ]
    [ "${lines[1]}"="Nodes: 2" ]

    run curl localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance

    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: false"* ]]
}

@test "Maintenance State: Setting maintenance state" {
    # start_docker will start a given number of engines:
    start_docker 3

    # The address of those engines is available through $HOSTS:
    run docker -H ${HOSTS[0]} info
    # "run" automatically populates $status, $output and $lines.
    # Please refer to bats documentation to find out more.

    [ "$status" -eq 0 ]

    # swarm_manage will start the swarm manager, preconfigured with all engines
    # previously started by start_engine:
    swarm_manage

    # You can talk with said manager by using the docker_swarm helper:
    run docker_swarm info
    [ "$status" -eq 0 ]
    [ "${lines[1]}"="Nodes: 2" ]

    run curl localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance
    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: false"* ]]

    run curl -X POST localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance/start

    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: true"* ]]

    run curl localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance
    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: true"* ]]
}

@test "Maintenance State: Verify nodes are not scheduled on nodes in maintenance state" {
	# TODO: What scenario do we chose? Take a preference for a node, verify it *not* be scheduled on node
	# in maintenance mode?
    start_docker_with_busybox 2
    swarm_manage

    # TODO: limit amount of containers started
    run docker_swarm run --name c1 -e constraint:node==node-1 -d busybox:latest sh
    [ "$status" -eq 0 ]
    run docker_swarm run --name c2 -e affinity:container==c1 -d busybox:latest sh
    [ "$status" -eq 0 ]
    # 116 failed with:
    # time="2016-04-15T00:02:18Z" level=error msg="HTTP error:
    # Unable to find a node that satisfies the following conditions \n[node==node-1 (soft=false)]" status=500

    run docker_swarm inspect c1
    # FIXME: This will help debugging the failing test.
    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *'"Name": "node-1"'* ]]

    run docker_swarm inspect c2
    [ "$status" -eq 0 ]
    [[ "${output}" == *'"Name": "node-1"'* ]]

    # Set node-1 into maintenance mode, we expect no new nodes to be scheduled on them, no matter their affinity for it.
    run curl localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance
    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: false"* ]]

    run curl -X POST localhost:${SWARM_BASE_PORT}/engines/node-1/maintenance/start

    echo $output
    [ "$status" -eq 0 ]
    [[ "${output}" == *"Engine maintenance status: true"* ]]

    # TODO: limit amount of containers started
    run docker_swarm run --name c3 -e affinity:container==~c2 -d busybox:latest sh
    [ "$status" -eq 0 ]

    run docker_swarm inspect c3
    [ "$status" -eq 0 ]
    [[ "${output}" != *'"Name": "node-1"'* ]]
}
