#!/usr/bin/env bats

load discovery_helpers

# Address on which the store will listen (random port between 8000 and 9000).
STORE_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 9000 ))

# Discovery parameter for Swarm
DISCOVERY="etcd://${STORE_HOST}/test"

# Container name for integration test
CONTAINER_NAME=swarm_etcd

function start_store() {
	docker_host run -p $STORE_HOST:4001 \
		--name=$CONTAINER_NAME -d \
		quay.io/coreos/etcd:v2.0.11 \
		--listen-client-urls=http://0.0.0.0:2379,http://0.0.0.0:4001 \
		--advertise-client-urls=http://$STORE_HOST
}

function stop_store() {
	docker_host rm -f -v $CONTAINER_NAME
}

function teardown() {
	swarm_manage_cleanup
	swarm_join_cleanup
	stop_docker
	stop_store
}

@test "etcd discovery: recover engines" {
	# The goal of this test is to ensure swarm can see engines that joined
	# while the manager was stopped.

	# Start the store
	start_store

	docker_host ps -a
	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"

	# Then, start a manager and ensure it sees all the engines.
	swarm_manage "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}

@test "etcd discovery: watch for changes" {
	# The goal of this test is to ensure swarm can see new nodes as they join
	# the cluster.
	start_store

	# Start a manager with no engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add engines to the cluster and make sure it's picked up by swarm.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}

@test "etcd discovery: node removal" {
	# The goal of this test is to ensure swarm can detect engines that
	# are removed from the discovery and refresh info accordingly

	# Start the store
	start_store

	# Start a manager with no engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add Engines to the cluster and make sure it's picked by swarm
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info

	# Removes all the swarm agents
	swarm_join_cleanup

	# Check if previously registered engines are all gone
	retry 15 1 discovery_check_swarm_info 0

	# Check that we can add instances back to the cluster
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info 2
}

@test "etcd discovery: failure" {
	# The goal of this test is to simulate a store failure and ensure discovery
	# is resilient to it.

	# At this point, the store is not yet started.
	
	# Start 2 engines and join the cluster. They should keep retrying
	start_docker 2
	swarm_join "$DISCOVERY"

	# Start a manager. It should keep retrying
	swarm_manage "$DISCOVERY"

	# Now start the store
	start_store

	# After a while, `join` and `manage` should reach the store.
	retry 5 1 discovery_check_swarm_list "$DISCOVERY"
	retry 5 1 discovery_check_swarm_info
}
