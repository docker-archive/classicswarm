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

	test_file="/bin/busybox"
	# create a temporary destination directory
	temp_dest=`mktemp -d`

	# create the container
	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container is up and no comming file
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]

	# grab the checksum of the test file inside the container.
	# FIXME: if issue #658 solved, use 'exec' instead of 'exec -i'
	run docker_swarm exec -i test_container md5sum $test_file
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 1 ]

	# get the checksum number
	container_checksum=`echo ${lines[0]} | awk '{print $1}'`

	# host file
	host_file=$temp_dest/`basename $test_file`
	[ ! -f $host_file ]

	# copy the test file from the container to the host.
	run docker_swarm cp test_container:$test_file $temp_dest
	[ "$status" -eq 0 ]
	[ -f $host_file ]

	# compute the checksum of the copied file.
	run md5sum $host_file
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 1 ]
	host_checksum=`echo ${lines[0]} | awk '{print $1}'`

	# Verify that they match.
	[[ "${container_checksum}" == "${host_checksum}" ]]
	# after ok, remove temp directory and file 
	rm -rf $temp_dest
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

	temp_file_name=$(mktemp)
	# make sure container exists 
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	
	# export, container->tar
	docker_swarm export test_container > $temp_file_name

	# verify: exported file exists, not empty and is tar file 
	[ -s $temp_file_name ]
	run file $temp_file_name
	[ "$status" -eq 0 ]
	echo ${lines[0]}
	echo $output
	[[ "$output" == *"tar archive"* ]]
	
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
	[[ "${output}" == *"busybox"* ]]

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

	temp_file_name=$(mktemp)
	temp_file_name_o=$(mktemp)
	# make sure busybox image exists
	run docker_swarm images 
	[ "$status" -eq 0 ]
	[[ "${output}" == *"busybox"* ]]

	# save >, image->tar
	docker_swarm save busybox > $temp_file_name
	# save -o, image->tar
	docker_swarm save -o $temp_file_name_o busybox
	
	# saved image file exists, not empty and is tar file 
	[ -s $temp_file_name ]
	run file $temp_file_name
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tar archive"* ]]

	[ -s $temp_file_name_o ]
	run file $temp_file_name_o
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tar archive"* ]]

	# after ok, delete saved tar file
	rm -f $temp_file_name
	rm -f $temp_file_name_o
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
