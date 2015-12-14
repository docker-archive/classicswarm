#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

# Address on which the store will listen (random port between 8000 and 9000).
STORE_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 7000 ))

# Discovery parameter for Swarm
DISCOVERY="zk://${STORE_HOST}/test"

# Container name for integration test
CONTAINER_NAME=swarm_integration_zk

function start_store() {
	docker_host run --name $CONTAINER_NAME -p $STORE_HOST:2181 -d dnephin/docker-zookeeper:3.4.6
}

function stop_store() {
	docker_host rm -f -v $CONTAINER_NAME
}

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
	stop_store
}

@test "mesos - zookeeper" {
	start_store
	start_docker 2
	start_mesos_zk $DISCOVERY

	swarm_manage --cluster-driver mesos-experimental $DISCOVERY

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *'Offers: 2'* ]]
}
