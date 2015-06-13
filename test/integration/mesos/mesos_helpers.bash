#!/bin/bash

load ../helpers

MESOS_IMAGE=jimenez/mesos-dev:clang
MESOS_MASTER_PORT=$(( ( RANDOM % 1000 )  + 10000 ))

# Start mesos master and slave.
function start_mesos() {
	MESOS_MASTER=$(
	docker_host run -d --name mesos-master --net=host \
	$MESOS_IMAGE /mesos/build/bin/mesos-master.sh --ip=127.0.0.1 --work_dir=/ --registry=in_memory --port=$MESOS_MASTER_PORT
		)

	retry 10 1 eval "docker_host ps | grep 'mesos-master'"

	MESOS_SLAVE=$(
	docker_host run --privileged -d --name mesos-slave --volumes-from node-0 -e DOCKER_HOST="${HOSTS[0]}" -v /sys/fs/cgroup:/sys/fs/cgroup --net=host \
	$MESOS_IMAGE /mesos/build/bin/mesos-slave.sh --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:${PORTS[0]}" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + 1))
	       )
	retry 10 1 eval "docker_host ps | grep 'mesos-slave'"
}

# Start the swarm manager in background.
function swarm_manage_mesos() {
	"$SWARM_BINARY" -l debug manage -H "$SWARM_HOST" --cluster-driver mesos-experimental --cluster-opt mesos.user=daemon 127.0.0.1:$MESOS_MASTER_PORT &
	SWARM_PID=$!
	retry 10 1 eval "docker_swarm info | grep 'Offers: 1'"
}

# Stop mesos master and slave.
function stop_mesos() {
	echo "Stopping $DOCKER_CONTAINER"
	docker_host rm -f -v $DOCKER_CONTAINER > /dev/null;
	echo "Stopping $MESOS_MASTER"
	docker_host rm -f -v $MESOS_MASTER > /dev/null;
	echo "Stopping $MESOS_SLAVE"
	docker_host rm -f -v $MESOS_SLAVE > /dev/null;
}
