#!/bin/bash

# /test/integration/helpers should be loaded before loading this file.

export SWARM_MESOS_TASK_TIMEOUT=30s
export SWARM_MESOS_USER=root

MESOS_IMAGE=dockerswarm/mesos:0.26.0
MESOS_MASTER_PORT=$(( ( RANDOM % 1000 )  + 10000 ))

# Start mesos master and agent.
function start_mesos() {
	local current=${#DOCKER_CONTAINERS[@]}
	MESOS_MASTER=$(
	docker_host run -d --name mesos-master --net=host \
	$MESOS_IMAGE mesos-master --ip=127.0.0.1 --work_dir=/ --registry=in_memory --port=$MESOS_MASTER_PORT
		)

	retry 10 1 eval "docker_host ps | grep 'mesos-master'"
	for ((i=0; i < current; i++)); do
	    local docker_port=$(echo ${HOSTS[$i]} | cut -d: -f2)
	    MESOS_AGENTS[$i]=$(
		docker_host run --privileged -d --name mesos-slave-$i --volumes-from node-$i -v /sys/fs/cgroup:/sys/fs/cgroup --net=host -u root \
		$MESOS_IMAGE mesos-slave --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:$docker_port" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + (1 + $i))) --docker=/usr/local/bin/docker
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
	    MESOS_AGENTS[$i]=$(
		docker_host run --privileged -d --name mesos-slave-$i --volumes-from node-$i -v /sys/fs/cgroup:/sys/fs/cgroup --net=host -u root \
		$MESOS_IMAGE mesos-slave --master=127.0.0.1:$MESOS_MASTER_PORT --containerizers=docker --attributes="docker_port:$docker_port" --hostname=127.0.0.1 --port=$(($MESOS_MASTER_PORT + (1 + $i))) --docker=/usr/local/bin/docker
			)
	    retry 10 1 eval "docker_host ps | grep 'mesos-slave-$i'"
	done
}

# Stop mesos master and agent
function stop_mesos() {
	echo "Stopping $MESOS_MASTER"
	docker_host rm -f -v $MESOS_MASTER > /dev/null;
	for id in ${MESOS_AGENTS[@]}; do
	    echo "Stopping $id"
	    docker_host rm -f -v $id > /dev/null;
	done
}
