#!/usr/bin/env bats

load compose_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "mesos - docker-compose ps" {
	start_docker_with_busybox 2
	start_mesos
	swarm_manage --cluster-driver mesos-experimental 127.0.0.1:$MESOS_MASTER_PORT
	FILE=$TESTDATA/compose/simple-resource.yml
	
	docker-compose_swarm -f $FILE up -d

	run docker-compose_swarm -f $FILE ps -q
	[ "${#lines[@]}" -eq  2 ]
}

