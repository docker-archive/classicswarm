#!/usr/bin/env bats

load ../integration/helpers

NODES=10
CONTAINERS=100

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

# Start N containers on the cluster in parallel that will sleep randomly.
function start_containers {
	local n="$1"
	local pids
	# Run `docker run` in parallel...
	for ((i=0; i < $n; i++)); do
		# Each container will sleep between 0 and 15 seconds.
		# This is to simuate a bunch of events back to Swarm (containers
		# exiting).
		local wait_time=$(( RANDOM % 15 ))
		docker_swarm run -d busybox:latest sleep $wait_time &
		pids[$i]=$!
	done

	# ...and wait for them to finish.
	for pid in ${pids[@]}; do
		wait $pid
	done
}

# Waits for all containers to exit (not running) for up to $1 seconds.
function wait_containers_exit {
	local attempts=0
	local max_attempts=$1
	until [ $attempts -ge $max_attempts ]; do
		run docker_swarm ps -q
		[[ "$status" -eq 0 ]]
		if [ "${#lines[@]}" -eq 0 ] ; then
			break
		fi
		echo "Containers still running: $output" >&2
		sleep 1
	done
	[[ $attempts -lt $max_attempts ]]
}

@test "spawning $CONTAINERS containers on $NODES nodes" {
	# Start N engines.
	start_docker_with_busybox $NODES

	# Start the manager
	swarm_manage

	# Verify that Swarm can see all the nodes.
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "$output" =~ "Nodes: $NODES" ]]

	# Start the containers (random sleeps).
	start_containers $CONTAINERS

	# Make sure all containers were scheduled.
	run docker_swarm ps -qa
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq  $CONTAINERS ]

	# Now wait for all those random sleeps to exit for up to 20 seconds.
	# This is to ensure Swarm is correctly keeping track of the container
	# status.
	wait_containers_exit 20

	# Make sure every node was utilized and the load was spread evenly (we are
	# using the spread strategy).
	run docker_swarm info
	# containers_per_node is the number of containers we should see on every
	# node. For instance, if we scheduled 10 containers on 2 nodes, then we
	# should have 5 containers per node.
	local containers_per_node=$(( $CONTAINERS / $NODES ))
	# Check how many nodes have the expected "containers_per_node" containers
	# count. It should equal to the total number of nodes.
	num_nodes=`echo $output | grep -o -e "Containers: ${containers_per_node} " | wc -l`
	[ "$num_nodes" -eq "$NODES" ]
}
