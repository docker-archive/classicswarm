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
