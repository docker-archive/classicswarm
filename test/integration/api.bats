#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

# FIXME
@test "docker attach" {
	skip
}

@test "docker build" {
      start_docker 3
      swarm_manage
      
      run docker_swarm images
      [ "$status" -eq 0 ]
      [ "${#lines[@]}" -eq 1 ]

      run docker_swarm build -t test $BATS_TEST_DIRNAME/testdata/build
      [ "$status" -eq 0 ]

      run docker_swarm images
      [ "$status" -eq 0 ]
      [ "${#lines[@]}" -eq 3 ]
}

# FIXME
@test "docker commit" {
	skip
}

# FIXME
@test "docker cp" {
	skip
}

@test "docker create" {
	start_docker 3
	swarm_manage

	# make sure no contaienr exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# create
	run docker_swarm create --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# verify, created container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
}

# FIXME
@test "docker diff" {
	skip
}

# FIXME
@test "docker events" {
	skip
}

# FIXME
@test "docker exec" {
	skip
}

# FIXME
@test "docker export" {
	skip
}

# FIXME
@test "docker history" {
	skip
}

# FIXME
@test "docker images" {
	skip
}

# FIXME
@test "docker import" {
	skip
}

@test "docker info" {
	start_docker 3
	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${lines[3]}" == *"Nodes: 3" ]]
}

# FIXME
@test "docker inspect" {
	skip
}

@test "docker kill" {
	start_docker 3
	swarm_manage
	# run 
	run docker_swarm run -d --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# make sure container is up before killing
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# kill
	run docker_swarm kill test_container
	[ "$status" -eq 0 ]
	# verify
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Exited"* ]]
}

# FIXME
@test "docker load" {
	skip
}

# FIXME
@test "docker login" {
	skip
}

# FIXME
@test "docker logout" {
	skip
}

# FIXME
@test "docker logs" {
	skip
}

# FIXME
@test "docker port" {
	skip
}

# FIXME
@test "docker pause" {
	skip
}

@test "docker ps -n" {
	start_docker 1
	swarm_manage
	run docker_swarm run -d busybox sleep 42
	run docker_swarm run -d busybox false
	run docker_swarm ps -n 3
	# Non-running containers should be included in ps -n
	[ "${#lines[@]}" -eq  3 ]

	run docker_swarm run -d busybox true
	run docker_swarm ps -n 3
	[ "${#lines[@]}" -eq  4 ]

	run docker_swarm run -d busybox true
	run docker_swarm ps -n 3
	[ "${#lines[@]}" -eq  4 ]
}

@test "docker ps -l" {
	start_docker 1
	swarm_manage
	run docker_swarm run -d busybox sleep 42
	sleep 1 #sleep so the 2 containers don't start at the same second
	run docker_swarm run -d busybox true
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq  2 ]
	# Last container should be "true", even though it's stopped.
	[[ "${lines[1]}" == *"true"* ]]

	sleep 1 #sleep so the container doesn't start at the same second as 'busybox true'
	run docker_swarm run -d busybox false
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"false"* ]]
}

# FIXME
@test "docker pull" {
	skip
}

# FIXME
@test "docker push" {
	skip
}

# FIXME
@test "docker rename" {
	skip
}

@test "docker restart" {
	start_docker 3
	swarm_manage
	# run 
	run docker_swarm run -d --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# restart
	run docker_swarm restart test_container
	[ "$status" -eq 0 ]
	# verify
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]
}

# FIXME
@test "docker rm" {
	skip
}

# FIXME
@test "docker rmi" {
	skip
}

# FIXME
@test "docker run" {
	skip
}

# FIXME
@test "docker save" {
	skip
}

# FIXME
@test "docker search" {
	skip
}

@test "docker start" {
	start_docker 3 
	swarm_manage
	# create
	run docker_swarm create --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# make sure created container exists
	# new created container has no status
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# start
	run docker_swarm start test_container
	[ "$status" -eq 0 ]
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" ==  *"Up"* ]]
}

# FIXME
@test "docker stats" {
	skip
}

@test "docker stop" {
	start_docker 3
	swarm_manage
	# run 
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container is up before stop
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# stop
	run docker_swarm stop test_container
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Exited"* ]]
}

# FIXME
@test "docker tag" {
	skip
}

# FIXME
@test "docker top" {
	skip
}

# FIXME
@test "docker unpause" {
	skip
}

# FIXME
@test "docker version" {
	skip
}

# FIXME
@test "docker wait" {
	skip
}
