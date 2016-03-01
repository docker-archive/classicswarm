#!/usr/bin/env bats

load discovery_helpers

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
	swarm_join_cleanup
	stop_docker
	stop_store
}

@test "zk discovery: recover engines" {
	# The goal of this test is to ensure swarm can see engines that joined
	# while the manager was stopped.

	# Start the store
	start_store

	# Start 2 engines and make them join the cluster.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 10 1 discovery_check_swarm_list "$DISCOVERY"

	# Then, start a manager and ensure it sees all the engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info
}

@test "zk discovery: watch for changes" {
	# The goal of this test is to ensure swarm can see new nodes as they join
	# the cluster.
	start_store

	# Start a manager with no engines.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add engines to the cluster and make sure it's picked up by swarm.
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 10 1 discovery_check_swarm_list "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info
}

@test "zk discovery: node removal" {
	# The goal of this test is to ensure swarm can detect engines that
	# are removed from the discovery and refresh info accordingly

	# Start the store
	start_store

	# Start 2 engines and make them join the cluster.
	swarm_manage "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Add Engines to the cluster and make sure it's picked by swarm
	start_docker 2
	swarm_join "$DISCOVERY"
	retry 10 1 discovery_check_swarm_list "$DISCOVERY"
	retry 10 1 discovery_check_swarm_info

	# Removes all the swarm agents
	swarm_join_cleanup

	# Check if previously registered engines are all gone
	retry 20 1 discovery_check_swarm_info 0

	# Check that we can add instances back to the cluster
	start_docker 2
	swarm_join "$DISCOVERY"
	run docker_swarm info
	echo $output
	retry 10 1 discovery_check_swarm_info 2
}

@test "zk discovery: failure" {
	# The goal of this test is to simulate a store failure and ensure discovery
	# is resilient to it.

	# At this point, the store is not yet started.
	
	# Start 2 engines and join the cluster. They should keep retrying
	start_docker 2
	swarm_join "$DISCOVERY"

	# Start a manager. It should keep retrying
	swarm_manage_no_wait "$DISCOVERY"

	# Now start the store
	start_store

	# After a while, `join` and `manage` should reach the store.
	retry 20 1 discovery_check_swarm_list "$DISCOVERY"
	run docker_swarm info
	echo $output
	retry 20 1 discovery_check_swarm_info
}
