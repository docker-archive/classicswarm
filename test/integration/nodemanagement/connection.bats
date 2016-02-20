#!/usr/bin/env bats

load ../helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "Swarm not leak tcp connections" {
	# Start engine with busybox image
	start_docker_with_busybox 2
	# Start swarm and check it can reach the node
	swarm_manage --engine-refresh-min-interval "20s" --engine-refresh-max-interval "20s" --engine-failure-retry 20 "${HOSTS[0]},${HOSTS[1]}"
	eval "docker_swarm info | grep -q -i 'Nodes: 2'"

	# create busybox with host network so that we can get netstat
	run docker_swarm run -itd --name=busybox0 --net=host -e constraint:node==node-0 busybox sh
	[ "$status" -eq 0 ]
	run docker_swarm run -itd --name=busybox1 --net=host -e constraint:node==node-1 busybox sh
	[ "$status" -eq 0 ]

	# run most common container operations
	for((i=0; i<30; i++)); do
		# test postContainerCreate
		docker_swarm run --name="hello$i" hello-world
		# test getContainerJSON
		docker_swarm inspect "hello$i"
		# test proxyContainer
		docker_swarm logs "hello$i"
		# test proxyContainerAndForceRefresh
		docker_swarm stop "hello$i"
	done

	# get connection count
	count0=$(docker_swarm exec busybox0 netstat -an | grep "${HOSTS[0]}" | grep -i "ESTABLISHED" | wc -l)
	count1=$(docker_swarm exec busybox1 netstat -an | grep "${HOSTS[1]}" | grep -i "ESTABLISHED" | wc -l)
	[[ "$count0" -le 10 ]]
	[[ "$count1" -le 10 ]]
}

