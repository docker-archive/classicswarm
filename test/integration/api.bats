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

@test "docker commit" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exists
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" ==  *"test_container"* ]]

	# no comming name before commit 
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" != *"commit_image_busybox"* ]]

	# commit container
	run docker_swarm commit test_container commit_image_busybox
	[ "$status" -eq 0 ]

	# verify after commit 
	run docker_swarm images
	[ "$status" -eq 0 ]
	[[ "${output}" == *"commit_image_busybox"* ]]
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
	run docker_swarm exec test_container md5sum $test_file
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

@test "docker exec" {
	start_docker 3
	swarm_manage
	run docker_swarm run -d --name test_container busybox sleep 100
	[ "$status" -eq 0 ]

	# make sure container is up and not paused
	run docker_swarm ps -l
	[ "${#lines[@]}" -eq 2 ]
	[[ "${lines[1]}" == *"test_container"* ]]
	[[ "${lines[1]}" == *"Up"* ]]
	[[ "${lines[1]}" != *"Paused"* ]]	

	run docker_swarm exec test_container ls
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -ge 2 ]
	[[ "${lines[0]}" == *"bin"* ]]
	[[ "${lines[1]}" == *"dev"* ]]
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

@test "docker images" {
	start_docker 3
	swarm_manage

	# no image exist
	run docker_swarm images -q 
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]
	# make sure every node has no image
	for((i=0; i<3; i++)); do
		run docker_swarm images --filter node=node-$i -q
		[ "$status" -eq 0 ]
		[ "${#lines[@]}" -eq 0 ]
	done

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

@test "docker load" {
	# temp file for saving image
	IMAGE_FILE=$(mktemp)

	# create a tar file
	docker pull busybox:latest
	docker save -o $IMAGE_FILE busybox:latest

	start_docker 2
	swarm_manage

	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq  0 ]

	run docker_swarm load -i $IMAGE_FILE
	[ "$status" -eq 0 ]
	
	# check node0
	run docker -H  ${HOSTS[0]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	# check node1
	run docker -H  ${HOSTS[1]} images
	[ "${#lines[@]}" -eq  2 ]
	[[ "${lines[1]}" == *"busybox"* ]]

	rm -f $IMAGE_FILE
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

@test "docker pull" {
	start_docker 3
	swarm_manage

	# make sure no image exists
	run docker_swarm images -q
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -eq 0 ]

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

@test "docker rename" {
	start_docker 3
	swarm_manage

	run docker_swarm run -d --name test_container busybox sleep 500
	[ "$status" -eq 0 ]

	run docker_swarm run -d --name another_container busybox sleep 500
	[ "$status" -eq 0 ]

	# make sure container exist
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"test_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"rename_container"* ]]

	# rename container, conflict and fail
	run docker_swarm rename test_container another_container
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Error when allocating new name: Conflict."* ]]

	# rename container, sucessful
	run docker_swarm rename test_container rename_container
	[ "$status" -eq 0 ]

	# verify after, rename 
	run docker_swarm ps -a
	[ "${#lines[@]}" -eq 3 ]
	[[ "${output}" == *"rename_container"* ]]
	[[ "${output}" == *"another_container"* ]]
	[[ "${output}" != *"test_container"* ]]
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
	[[ "${output}" == *"busybox"* ]]
	[[ "${output}" != *"tag_busybox"* ]]

	# tag image
	run docker_swarm tag busybox tag_busybox:test
	[ "$status" -eq 0 ]

	# verify
	run docker_swarm images tag_busybox
	[ "$status" -eq 0 ]
	[[ "${output}" == *"tag_busybox"* ]]
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
