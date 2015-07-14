#!/bin/bash

load ../../helpers

MESOS_IMAGE=jimenez/mesos-dev:clang
MESOS_MASTER_PORT=$(( ( RANDOM % 1000 )  + 10000 ))

# Start mesos master and slave.
function start_mesos() {
	local current=${#DOCKER_CONTAINERS[@]}
	MESOS_MASTER=$(
	docker_host run -d --name mesos-master --net=host \
	$MESOS_IMAGE /mesos/build/bin/mesos-master.sh --ip=127.0.0.1 --work_dir=/ --registry=in_memory --port=$MESOS_MASTER_PORT
		)

	retry 10 1 eval "docker_host ps | grep 'mesos-master'"
	for ((i=0; i < current; i++)); do
	    local docker_port=$(echo ${HOSTS[$i]} | cut -d: -f2)
	    MESOS_SLAVES[$i]=$(
		docker_host run --privileged -d --name mesos-slave-$i --volumes-from node-$i -e DOCKER_HOST="${HOSTS[$i]}" -v /sys/fs/cgroup:/sys/fs/cgroup --net=host \
		$MESOS_IMAGE /mesos/build/bin/mesos-slave.sh --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:$docker_port" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + (1 + $i)))
		       )
	    retry 10 1 eval "docker_host ps | grep 'mesos-slave-$i'"
	done
}

# Start the swarm manager in background.
function swarm_manage_mesos() {
	local current=${#DOCKER_CONTAINERS[@]}
	"$SWARM_BINARY" -l debug manage -H "$SWARM_HOST" --cluster-driver mesos-experimental --cluster-opt mesos.user=daemon 127.0.0.1:$MESOS_MASTER_PORT &
	SWARM_PID=$!
	retry 10 1 eval "docker_swarm info | grep 'Offers: $current'"
}

# Stop mesos master and slave.
function stop_mesos() {
	echo "Stopping $MESOS_MASTER"
	docker_host rm -f -v $MESOS_MASTER > /dev/null;
	for id in ${MESOS_SLAVES[@]}; do
	    echo "Stopping $id"
	    docker_host rm -f -v $id > /dev/null;
	done
}
