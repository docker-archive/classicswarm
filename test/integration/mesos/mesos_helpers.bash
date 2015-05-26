#!/bin/bash

load ../helpers

MESOS_CLUSTER_ENTRYPOINT=${MESOS_CLUSTER_ENTRYPOINT:-0.0.0.0:5050}

# Start the swarm manager in background.
function swarm_manage_mesos() {
	"$SWARM_BINARY" -l debug manage -H "$SWARM_HOST" --cluster-driver mesos $MESOS_CLUSTER_ENTRYPOINT &
	SWARM_PID=$!
	wait_until_reachable "$SWARM_HOST"
}
