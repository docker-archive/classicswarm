#!/bin/bash

load ../../helpers

export SWARM_MESOS_TASK_TIMEOUT=30s
export SWARM_MESOS_USER=daemon

MESOS_IMAGE=dockerswarm/mesos:0.24.1
MESOS_MASTER_PORT=$(( ( RANDOM % 1000 )  + 10000 ))

# Start mesos master and slave.
function start_mesos() {
	local current=${#DOCKER_CONTAINERS[@]}
	MESOS_MASTER=$(
	docker_host run -d --name mesos-master --net=host \
	$MESOS_IMAGE mesos-master --ip=127.0.0.1 --work_dir=/ --registry=in_memory --port=$MESOS_MASTER_PORT
		)

	retry 10 1 eval "docker_host ps | grep 'mesos-master'"
	for ((i=0; i < current; i++)); do
	    local docker_port=$(echo ${HOSTS[$i]} | cut -d: -f2)
	    MESOS_SLAVES[$i]=$(
		docker_host run --privileged -d --name mesos-slave-$i --volumes-from node-$i -e DOCKER_HOST="${HOSTS[$i]}" -v /sys/fs/cgroup:/sys/fs/cgroup --net=host \
		$MESOS_IMAGE mesos-slave --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:$docker_port" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + (1 + $i))) --docker=/usr/local/bin/docker --executor_environment_variables="{\"DOCKER_HOST\":\"${HOSTS[$i]}\"}"
		       )
	    retry 10 1 eval "docker_host ps | grep 'mesos-slave-$i'"
	done
}

function start_mesos_zk() {
	local current=${#DOCKER_CONTAINERS[@]}
	MESOS_MASTER=$(
	docker_host run -d --name mesos-master --net=host \
	$MESOS_IMAGE mesos-master --ip=127.0.0.1 --work_dir=/var/lib/mesos --registry=in_memory --quorum=2 --zk=$1 --port=$MESOS_MASTER_PORT
		)

	retry 10 1 eval "docker_host ps | grep 'mesos-master'"
	for ((i=0; i < current; i++)); do
	    local docker_port=$(echo ${HOSTS[$i]} | cut -d: -f2)
	    MESOS_SLAVES[$i]=$(
		docker_host run --privileged -d --name mesos-slave-$i --volumes-from node-$i -e DOCKER_HOST="${HOSTS[$i]}" -v /sys/fs/cgroup:/sys/fs/cgroup --net=host \
		$MESOS_IMAGE mesos-slave --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:$docker_port" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + (1 + $i))) --docker=/usr/local/bin/docker --executor_environment_variables="{\"DOCKER_HOST\":\"${HOSTS[$i]}\"}"
		       )
	    retry 10 1 eval "docker_host ps | grep 'mesos-slave-$i'"
	done
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
