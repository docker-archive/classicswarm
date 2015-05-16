#!/usr/bin/env bats

load ../helpers

# Address on which Etcd will listen (random port between 9000 and 10,000).
ETCD_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 9000 ))

# Container name for integration test
CONTAINER_NAME=swarm_etcd

function start_etcd() {
	docker_host run -p $ETCD_HOST:4001 --name=$CONTAINER_NAME -d coreos/etcd
}

function stop_etcd() {
	docker_host rm -f -v $CONTAINER_NAME
}

function setup() {
	start_etcd
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	stop_etcd
}

@test "etcd discovery" {
	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "etcd://${ETCD_HOST}/test"

	# Start a manager and ensure it sees all the engines.
	swarm_manage "etcd://${ETCD_HOST}/test"
	check_swarm_nodes

	# Add another engine to the cluster and make sure it's picked up by swarm.
	start_docker 1
	swarm_join "etcd://${ETCD_HOST}/test"
	retry 30 1 check_swarm_nodes
}
