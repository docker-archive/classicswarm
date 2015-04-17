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

@test "docker cp" {
	start_docker 3
	swarm_manage
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]
	temp_file_name="/tmp/cp_file_$RANDOM"
	# make sure container is up and no comming file
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]
	[ ! -f $temp_file_name ]	

	# touch file for cp 
	run docker_swarm exec test_container touch $temp_file_name
	[ "$status" -eq 0 ]

	# cp and verify
	run docker_swarm cp test_container:$temp_file_name /tmp/
	[ "$status" -eq 0 ]

	# verify: cp file exists
	[ -f $temp_file_name ]
	# after ok, delete cp file
	rm -f $temp_file_name
}

# FIXME
@test "docker create" {
	skip
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

@test "docker export" {
	start_docker 3
	swarm_manage
	# run a container to export
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	temp_file_name="/tmp/export_file_$RANDOM.tar"
	# make sure container exists and no comming file
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]
	[ ! -f $temp_file_name ]

	# export, container->tar
	run docker_swarm export test_container > $temp_file_name
	[ "$status" -eq 0 ]

	# verify: exported file exists
	[ -f $temp_file_name ]
	# after ok, delete exported tar file
	rm -f $temp_file_name
}

@test "docker history" {
	start_docker 3
	swarm_manage

	# pull busybox image
	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	# make sure the image of busybox exists
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"busybox"* ]]

	# history
	run docker_swarm history busybox
	[ "$status" -eq 0 ]
	[[ "${lines[0]}" == *"CREATED BY"* ]]
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

# FIXME
@test "docker kill" {
	skip
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

# FIXME
@test "docker restart" {
	skip
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

@test "docker save" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	temp_file_name="/tmp/save_file_$RANDOM.tar"
	# make sure busybox image exists and no comming file in current path
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"busybox"* ]]
	[ ! -f $temp_file_name ]

	# save >, image->tar
	run docker_swarm save busybox > $temp_file_name
	[ "$status" -eq 0 ]

	# saved image file exists
	[ -f $temp_file_name ]
	# after ok, delete saved tar file
	rm -f $temp_file_name
}

@test "docker save -o" {
	start_docker 3
	swarm_manage

	run docker_swarm pull busybox
	[ "$status" -eq 0 ]

	temp_file_name="/tmp/save_o_file_$RANDOM.tar"
	# make sure busybox image exists and no comming file in current path
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[[ "${lines[*]}" == *"busybox"* ]]
	[ ! -f $temp_file_name ]

	# save -o, image->tar
	run docker_swarm save -o $temp_file_name busybox 
	[ "$status" -eq 0 ]

	# saved image file exists
	[ -f $temp_file_name ]
	# after ok, delete saved tar file
	rm -f $temp_file_name
}

# FIXME
@test "docker search" {
	skip
}

# FIXME
@test "docker start" {
	skip
}

# FIXME
@test "docker stats" {
	skip
}

# FIXME
@test "docker stop" {
	skip
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
