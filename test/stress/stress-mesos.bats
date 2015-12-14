#!/usr/bin/env bats

load ../integration/helpers
load ../integration/mesos/mesos_helpers

NODES=10
CONTAINERS=100

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "spawning $CONTAINERS containers on $NODES nodes" {
	start_docker_with_busybox $NODES
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT

	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${output}" == *"Offers: ${NODES}"* ]]

	for i in `seq 1 100`; do
	    docker_swarm run -d -m 20m busybox echo $i
	done

	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq  $CONTAINERS ]
}
