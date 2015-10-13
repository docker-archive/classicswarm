#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
	stop_store
}

# Address on which the store will listen (random port between 8000 and 9000).
STORE_HOST=127.0.0.1:$(( ( RANDOM % 1000 )  + 8000 ))

# Discovery parameter for Swarm
DISCOVERY="consul://${STORE_HOST}/test"

# Container name for integration test
CONTAINER_NAME=swarm_consul

function start_store() {
	docker_host run -v $(pwd)/discovery/consul/config:/config --name=$CONTAINER_NAME -h $CONTAINER_NAME -p $STORE_HOST:8500 -d progrium/consul -server -bootstrap-expect 1 -config-file=/config/consul.json
}

function stop_store() {
	docker_host rm -f -v $CONTAINER_NAME || true
}


@test "docker setup cluster-store" {
	start_store

	start_docker_with_busybox 1 --cluster-store $DISCOVERY --cluster-advertise $HOSTS[0]
	start_docker_with_busybox 1 --cluster-store $DISCOVERY --cluster-advertise $HOSTS[1]

	run docker_host -H ${HOSTS[0]} info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"$DISCOVERY"* ]]

	run docker_host -H ${HOSTS[1]} info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"$DISCOVERY"* ]]

	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Nodes: 2"* ]]
}
