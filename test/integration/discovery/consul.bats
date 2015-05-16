#!/usr/bin/env bats

load ../helpers

# Address on which Consul will listen (random port between 8000 and 9000).
CONSUL_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 8000 ))

# Container name for integration test
CONTAINER_NAME=swarm_consul

function start_consul() {
	docker_host run --name=$CONTAINER_NAME -h $CONTAINER_NAME -p $CONSUL_HOST:8500 -d progrium/consul -server -bootstrap-expect 1 -data-dir /test
}

function stop_consul() {
	docker_host rm -f -v $CONTAINER_NAME
}

function setup() {
	start_consul
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	stop_consul
}

@test "consul discovery" {
	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "consul://${CONSUL_HOST}/test"

	# Start a manager and ensure it sees all the engines.
	swarm_manage "consul://${CONSUL_HOST}/test"
	check_swarm_nodes

	# Add another engine to the cluster and make sure it's picked up by swarm.
	start_docker 1
	swarm_join "consul://${CONSUL_HOST}/test"
	retry 30 1 check_swarm_nodes
}
