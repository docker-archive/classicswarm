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

@test "Check container states" {
    #skip
    # create daemons
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name top busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	topConfig1Id=$output
	
	run notAuthorized $DOCKER_CONFIG2 top
	echo $status
	[ "$status" -eq 0 ]
	run notAuthorized $DOCKER_CONFIG2 $topConfig1Id
	[ "$status" -eq 0 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --name top busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	topConfig2Id=$output
	[[ "$topConfig1Id" != "$topConfig2Id" ]]

    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect top
    [ "$status" -eq 0 ]
    inspectConfig1=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect top
    [ "$status" -eq 0 ]
    [ "$inspectConfig1" = "$output" ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' top 
    [ "$status" -eq 0 ]
	[[ "$output" == "$topConfig1Id /top created" ]]	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 start top  
    [ "$status" -eq 0 ]
    [[ "$output" = "top" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' top 
    [ "$status" -eq 0 ]
	[[ "$output" == "$topConfig1Id /top running" ]]	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 pause top  
    [ "$status" -eq 0 ]
    [[ "$output" = "top" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' top 
    [ "$status" -eq 0 ]
	[[ "$output" == "$topConfig1Id /top paused" ]]	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 unpause top  
    [ "$status" -eq 0 ]
    [[ "$output" = "top" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' top 
    [ "$status" -eq 0 ]
	[[ "$output" == "$topConfig1Id /top running" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 stop top  
    [ "$status" -eq 0 ]
    [[ "$output" = "top" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' top 
    [ "$status" -eq 0 ]
	[[ "$output" == "$topConfig1Id /top exited" ]]	

    run checkInvariant
    [ $status = 0 ]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 rm -v top   
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
		
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 stop top   
    [ "$status" -eq 0 ]
    [[ "$output" = "top" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 rm -v top   
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]	
	
}





