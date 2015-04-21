#!/usr/bin/env bats

load helpers

function teardown() {
	swarm_manage_cleanup
	stop_docker
}

@test "docker attach" {
	start_docker 3
	swarm_manage

	# container run in background
	run docker_swarm run -d --name test_container busybox sleep 100
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
	[[ "${lines[1]}" ==  *"Up"* ]]

	# attach to running container
	run docker_swarm attach test_container
	[ "$status" -eq 0 ]
}

@test "docker attach through websocket" {
	CLIENT_API_VERSION="v1.17"
	start_docker 2
	swarm_manage

	#create a container
        run docker_swarm run -d --name test_container busybox sleep 1000

	# test attach-ws api
	# jimmyxian/centos7-wssh is an image with websocket CLI(WSSH) wirtten in Nodejs
	# if connected successfull, it returns two lines, "Session Open" and "Session Closed"
	# Note: with stdout=1&stdin=1&stream=1: it can be used as SSH
	URL="ws://${SWARM_HOST}/${CLIENT_API_VERSION}/containers/test_container/attach/ws?stderr=1"
	run docker run --rm --net=host jimmyxian/centos7-wssh wssh $URL
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[0]}" == *"Session Open"* ]]
	[[ "${lines[1]}" == *"Session Closed"* ]]
}

@test "docker build" {
      start_docker 3
      swarm_manage
      
      run docker_swarm images -q
      [ "$status" -eq 0 ]
      [ "${#lines[@]}" -eq 0 ]

      run docker_swarm build -t test $BATS_TEST_DIRNAME/testdata/build
      [ "$status" -eq 0 ]

      run docker_swarm images -q
      [ "$status" -eq 0 ]
      [ "${#lines[@]}" -eq 1 ]
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

@test "docker diff" {
	start_docker 3
	swarm_manage
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
	[[ "${lines[1]}" ==  *"Up"* ]]

	# no changs
	run docker_swarm diff test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	# make changes on container's filesystem
	run docker_swarm exec test_container touch /home/diff.txt
	[ "$status" -eq 0 ]
	# verify
	run docker_swarm diff test_container
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" ==  *"diff.txt"* ]]
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

@test "docker images" {
	start_docker 3
	swarm_manage

	# no image exist
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	[[ "${lines[0]}" == *"REPOSITORY"* ]]

	# pull image
	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# show all node images, including reduplicated
	run docker_swarm images
	[ "$status" -eq 0 ]
	# check pull busybox, if download sucessfully, the busybox have one tag(lastest) at least
	# if there are 3 nodes, the output lines of "docker images" are greater or equal 4(1 header + 3 busybox:latest)
	# so use -ge here, the following(pull/tag) is the same reason
	[ "${#lines[@]}" -ge 4 ]
	# Every line should contain "busybox" exclude the first head line 
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done
}

@test "docker images --filter node=<Node name>" {
	start_docker 3
	swarm_manage

	# no image exist
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	[[ "${lines[0]}" == *"REPOSITORY"* ]]

	# make sure every node has no image
	for((i=0;i<3;i++)); do
		run docker_swarm images --filter node=node-$i
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 1 ]
		[[ "${lines[0]}" == *"REPOSITORY"* ]]
	done

	# pull image
	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure images have pulled
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 4 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	# verify
	for((i=0; i<3; i++)); do 
		run docker_swarm images --filter node=node-$i
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -ge 2 ]
		[[ "${lines[1]}" == *"busybox"* ]]
	done
}

# FIXME
@test "docker import" {
	skip
}

@test "docker info" {
	start_docker 1 --label foo=bar
	swarm_manage
	run docker_swarm info
	[ "$status" -eq 0 ]
	[[ "${lines[3]}" == *"Nodes: 1" ]]
	[[ "${output}" == *"└ Labels:"*"foo=bar"* ]]
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

@test "docker logs" {
	start_docker 3
	swarm_manage

	# run a container with echo command
	run docker_swarm run -d --name test_container busybox /bin/sh -c "echo hello world; echo hello docker; echo hello swarm"
	[ "$status" -eq 0 ]

	# make sure container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# verify
	run docker_swarm logs test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 3 ]
	[[ "${lines[0]}" ==  *"hello world"* ]]
	[[ "${lines[1]}" ==  *"hello docker"* ]]
	[[ "${lines[2]}" ==  *"hello swarm"* ]]
}

@test "docker port" {
	start_docker 3
	swarm_manage
	run docker_swarm run -d -p 8000 --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# port verify
	run docker_swarm port test_container
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"8000"* ]]
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

@test "docker pull" {
	start_docker 3
	swarm_manage

	# make sure no image exists
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

	for host in ${HOSTS[@]}; do
		run docker -H $host images -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# swarm verify
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 4 ]
	# every line should contain "busybox" exclude the first head line 
	for((i=1; i<${#lines[@]}; i++)); do
		[[ "${lines[i]}" == *"busybox"* ]]
	done

	# node verify
	for host in ${HOSTS[@]}; do
		run docker -H $host images
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -ge 2 ]
		[[ "${lines[1]}" == *"busybox"* ]]
	done
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

@test "docker rm" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox
	[ "$status" -eq 0 ]

	# make sure container exsists and is exited
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Exited"* ]]

	run docker_swarm rm test_container
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq 0 ]
}

@test "docker rm -f" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exsists and is up
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# rm, remove a running container, return error
	run docker_swarm rm test_container
	[ "$status" -ne 0 ]

	# rm -f, remove a running container
	run docker_swarm rm -f test_container
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm ps -aq
	[ "${#lines[@]}" -eq 0 ]
}

@test "docker rmi" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure image exists
	# swarm check image
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]
	# node check image
	for host in ${HOSTS[@]}; do
		run docker -H $host images
		[ "$status" -eq 0 ]
		[[ "${output}" == *"busybox"* ]]
	done

	# this test presupposition: do not run image
	run docker_swarm rmi busybox
	[ "$status" -eq 0 ]

	# swarm verify
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
	# node verify
	for host in ${HOSTS[@]}; do
		run docker -H $host images -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done
}

@test "docker run" {
	start_docker 3
	swarm_manage

	# make sure no container exist
	run docker_swarm ps -qa
	[ "${#lines[@]}" -eq 0 ]

	# run
	run docker_swarm run -d --name test_container busybox sleep 100
	[ "$status" -eq 0 ]

	# verify, container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${output}" == *"test_container"* ]]
	[[ "${output}" == *"Up"* ]]
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

@test "docker tag" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure the image of busybox exists 
	# the comming image of tag_busybox not exsit
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${lines[*]}" == *"busybox"* ]]
	[[ "${lines[*]}" != *"tag_busybox"* ]]

	# tag image
	run docker_swarm tag busybox tag_busybox:test
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm images
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${lines[*]}" == *"tag_busybox"* ]]
}

@test "docker top" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]
	# make sure container is running
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	run docker_swarm top test_container
	[ "$status" -eq 0 ]
	[[ "${lines[0]}" == *"UID"* ]]
	[[ "${lines[0]}" == *"CMD"* ]]
	[[ "${lines[1]}" == *"sleep 500"* ]]
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
