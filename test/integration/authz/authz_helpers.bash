#!/bin/bash

#FiWare credentials

export TENANT1=$TENANT_NAME
#export KEYSTONE_IP='cloud.lab.fi-ware.org:4730'

#export DISCOVERY_FILE="/root/work/src/github.com/docker/swarm/my_cluster"
#export DISCOVERY="--multiTenant file://$DISCOVERY_FILE"

export DOCKER_IMAGE=${DOCKER_IMAGE:-dockerswarm/dind}
export DOCKER_VERSION=${DOCKER_VERSION:-1.9.0}
export STORAGE_DRIVER='aufs'

# Root directory of integration tests.
INTEGRATION_ROOT=$(dirname "$(readlink -f "$BASH_SOURCE")")

# Root directory of the repository.
SWARM_ROOT=${SWARM_ROOT:-$(cd "$INTEGRATION_ROOT/../../.."; pwd -P)}

# Path of the Swarm binary.
SWARM_BINARY=${SWARM_BINARY:-${SWARM_ROOT}/swarm}

load ../helpers

# Waits until the given docker engine API becomes reachable.
function wait_until_reachable() {
	retry 10 1 docker -H $1 ps
}

function loginToKeystoneTenant1(){
	export $TENANT_NAME=$TENANT1
	$INTEGRATION_ROOT/../../tools/set_docker_conf.bash
}

function getAnotherTenant(){
    OUTPUT="$(get_tenants)"
    IFS=","
    for tenant in $OUTPUT;
    do
        tenant=`echo $tenant | xargs`
        if [[ "$tenant" != "$1" ]] ; then
            echo $tenant
            return
        fi
    done
}

function loginToKeystoneTenant2(){
	export TENANT_NAME=$(getAnotherTenant "$TENANT_NAME")
	$INTEGRATION_ROOT/../../tools/set_docker_conf.bash
}

# Start the swarm manager in background.
function swarm_manage_multy_tenant() {
	local discovery
	discovery=`join , ${HOSTS[@]}`
	
	local i=${#SWARM_MANAGE_PID[@]}
	local port=$(($SWARM_BASE_PORT + $i))
	local host=127.0.0.1:$port
	
	"$SWARM_BINARY" -l debug manage -H "$host" --heartbeat=1s --multiTenant $discovery &
	SWARM_MANAGE_PID[$i]=$!
	SWARM_HOSTS[$i]=$host
	wait_until_reachable "$host"
}
