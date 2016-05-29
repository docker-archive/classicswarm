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

@test "Check run and inspect" {
    #skip
	# run non daemons
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1  run --name  busy1 busybox
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect busy1
    [ "$status" -eq 0 ]
	inspectConfig1=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect -f '{{.Name}} {{.State.Status}}' busy1 
    [ "$status" -eq 0 ]
	[[ "$output" == "/busy1 exited" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect -f '{{.Id}}' busy1 
    [ "$status" -eq 0 ]
	config1Busy1Id=$output

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect busy1
    [ "$status" -eq 0 ]
	[[ "$output" == "$inspectConfig1" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect $config1Busy1Id
    [ "$status" -eq 0 ]
	[[ "$output" == "$inspectConfig1" ]]
   
    # same name different tenant
	run notAuthorized $DOCKER_CONFIG2 busy1
	[ "$status" -eq 0 ]
	run notAuthorized $DOCKER_CONFIG2 $config1Busy1Id
	[ "$status" -eq 0 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2  run --name  busy1 busybox
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect busy1
    [ "$status" -eq 0 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect -f '{{.Id}}' busy1 
    [ "$status" -eq 0 ]
	config2Busy1Id=$output
	[[ config1Busy1Id != config2Busy1Id ]]


    # run daemons
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --name loop1 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	loop1Config1Id=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect loop1
    [ "$status" -eq 0 ]
    inspectConfig1=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect loop1
    [ "$status" -eq 0 ]
    [ "$inspectConfig1" = "$output" ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect $loop1Config1Id
    [ "$status" -eq 0 ]
    [ "$inspectConfig1" = "$output" ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' loop1 
    [ "$status" -eq 0 ]
	[[ "$output" == "$loop1Config1Id /loop1 running" ]]
 
    # same name different tenant
	run notAuthorized $DOCKER_CONFIG2 loop1
	[ "$status" -eq 0 ]
	run notAuthorized $DOCKER_CONFIG2 $loop1Config1Id
	[ "$status" -eq 0 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2  rm -f loop1 
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2  rm -f $loop1Config1Id 
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
 
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --name loop1 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	loop1Config2Id=$output
	[[ "$loop1Config1Id" != "$loop1Config2Id" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect -f '{{.Id}} {{.Name}} {{.State.Status}}' loop1 
    [ "$status" -eq 0 ]
	[[ "$output" == "$loop1Config2Id /loop1 running" ]]
   

    # different user of tenant
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 run -d --name loop2 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect loop2
    [ "$status" -eq 0 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect loop2
    [ "$status" -eq 0 ]

    run checkInvariant
    [ $status = 0 ]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 rm -fv busy1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 rm -fv loop1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 rm -fv loop2
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 rm -fv loop1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 rm -fv busy1
	[ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]

}





