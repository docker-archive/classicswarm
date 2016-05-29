#!/usr/bin/env bats

######################################################################################
# cli.bats tests multi-tenant swarm
# The following environment variables are required
# SWARM_HOST The IP and Port of the SWARM HOST.  It is in form of tcp://<ip>:<port>
# DOCKER_CONFIG1  Directory where the docker config.json file for the Tenant 1, User 1
# DOCKER_CONFIG2  Directory where the docker config.json file for the Tenant 2, User 2
# DOCKER_CONFIG3  Directory where the docker config.json file for the Tenant 1, User 3
#
# Notes on test logic
#  Before each test all containers are remove from the Tenant 1 and Tenant 2 (see setup()))
#  After each test the invariant is checked (checkInvariant()).  checkInvariant checks
#  Tenant 1 and Tenant 2 containers are different from each other and that User 1 and User 3
#  containers are the same.
######################################################################################
  

load cli_helpers

@test "Check affinity:container==" {
    #skip
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --name frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	frontendid=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d -e affinity:container==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer1=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d -e affinity:container==$frontendid busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer2=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 run -d -e affinity:container==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer3=$output	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect --format={{.Node.Name}} $frontendid
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	nodeName="$output"
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect --format={{.Node.Name}} $refer1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect --format={{.Node.Name}} $refer2
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect --format={{.Node.Name}} $refer3
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
	#validate tenant isolation
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d -e affinity:container==frontend busybox true
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --name frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	frontendid=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d -e affinity:container==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer1=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect --format={{.Node.Name}} $frontendid
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	nodeName="$output"
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect --format={{.Node.Name}} $refer1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
		
    run checkInvariant
    [ $status = 0 ]  

}
@test "Check affinity:<label>==" {
    #skip
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --label com.example.type=frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	frontendid=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d -e affinity:com.example.type==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer1=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 run -d -e affinity:com.example.type==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer3=$output	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect --format={{.Node.Name}} $frontendid
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	nodeName="$output"
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect --format={{.Node.Name}} $refer1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect --format={{.Node.Name}} $refer3
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
	#validate tenant isolation
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d -e affinity:com.example.type==frontend busybox true
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --label com.example.type=frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	frontendid=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d -e affinity:com.example.type==frontend busybox true
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	refer1=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect --format={{.Node.Name}} $frontendid
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	nodeName="$output"
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect --format={{.Node.Name}} $refer1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $nodeName == "$output" ]]
		
    run checkInvariant
    [ $status = 0 ]  

}





