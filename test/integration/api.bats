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

# FIXME
@test "docker images" {
	skip
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

@test "docker inspect" {
	start_docker 3
	swarm_manage
	# run container
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exsists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	# inspect and verify 
	run docker_swarm inspect test_container
	[ "$status" -eq 0 ]
	[[ "${lines[1]}" == *"AppArmorProfile"* ]]
	# the specific information of swarm node
	[[ ${output} == *'"Node": {'* ]]
	[[ ${output} == *'"Name": "node-'* ]]
}

@test "docker inspect --format" {
	start_docker 3
	swarm_manage
	# run container
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exsists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]

	# inspect --format='{{.Config.Image}}', return one line: image name
	run docker_swarm inspect --format='{{.Config.Image}}' test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	[[ "${lines[0]}" == "busybox" ]]

	# inspect --format='{{.Node.IP}}', return one line: Node ip
	run docker_swarm inspect --format='{{.Node.IP}}' test_container
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 1 ]
	[[ "${lines[0]}" == "127.0.0.1" ]]
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

@test "docker pause" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	run docker_swarm pause test_container
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Paused"* ]]

	# if the state of the container is paused, it can't be removed(rm -f)	
	run docker_swarm unpause test_container
	[ "$status" -eq 0 ]
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

@test "docker rename" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exist
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[*]}" == *"test_container"* ]]
	[[ "${lines[*]}" != *"rename_container"* ]]

	# rename container
	run docker_swarm rename test_container rename_container
	[ "$status" -eq 0 ]

	# verify after rename 
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[*]}" == *"rename_container"* ]]
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

# FIXME
@test "docker tag" {
	skip
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

@test "docker unpause" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 1000
	[ "$status" -eq 0 ]

	# make sure container is up
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# pause
	run docker_swarm pause test_container
	[ "$status" -eq 0 ]
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Paused"* ]]

	# unpause
	run docker_swarm unpause test_container
	[ "$status" -eq 0 ]
	# verify
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]
	[[ "${lines[1]}" != *"Paused"* ]]
}

# FIXME
@test "docker version" {
	skip
}

# FIXME
@test "docker wait" {
	skip
}
