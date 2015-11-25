#!/usr/bin/env bats

load ../kubernetes_helpers

function teardown() {
	swarm_manage_cleanup
	stop_kubernetes
	stop_docker
}

@test "kubernetes - docker run" {
	start_docker_with_busybox 2
	start_kubernetes
	swarm_manage --cluster-driver kubernetes-experimental 127.0.0.1:$KUBERNETES_API_PORT

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	docker_swarm run -m 20m -d --name test_container busybox sleep 100

	# verify, container is running
	[ -n $(docker_swarm ps -q --filter=name=test_container --filter=status=running) ]

	# error check
	run docker_swarm run -m 20m -d 4e8aa3148a132f19ec560952231c4d39522043994df7d2dc239942c0f9424ebd
	[[ "${output}" == *"cannot specify 64-byte hexadecimal strings"* ]]
}

@test "kubernetes - docker run short lived" {
	start_docker_with_busybox 1
	start_kubernetes
	swarm_manage --cluster-driver kubernetes-experimental 127.0.0.1:$KUBERNETES_API_PORT

	# run
	run docker_swarm run -m 20m busybox true

	# make sure the container was started
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 1 ]
}

@test "kubernetes - docker run big" {
	start_docker_with_busybox 3
	start_kubernetes
	swarm_manage --cluster-driver kubernetes-experimental 127.0.0.1:$KUBERNETES_API_PORT

	for i in `seq 1 100`; do
	    docker_swarm run -d -m 20m busybox echo $i
	done

	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq  100 ]
}