#!/usr/bin/env bats

load helpers

# Address on which Zookeeper will listen (random port between 7000 and 8000).
ZK_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 7000 ))

# Container name for integration test
ZK_CONTAINER_NAME=swarm_integration_zk

function start_zk() {
	docker_host run --name $ZK_CONTAINER_NAME -p $ZK_HOST:2181 -d jplock/zookeeper
}

function stop_zk() {
	docker_host rm -f -v $ZK_CONTAINER_NAME > /dev/null
}

function teardown() {
	swarm_join_cleanup
	swarm_manage_cleanup
	stop_docker
	stop_zk
}

@test "zk discovery" {
	start_zk
	start_docker 1

	swarm_manage zk://${ZK_HOST}/test
	swarm_join   zk://${ZK_HOST}/test

	run docker_swarm info
	[[ "$output" == *"Nodes: 1"* ]]
}
