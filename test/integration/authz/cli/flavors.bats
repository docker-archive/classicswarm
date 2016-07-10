#!/usr/bin/env bats

######################################################################################
# cli.bats tests multi-tenant swarm
# The following environment variables are required
# SWARM_HOST The IP and Port of the SWARM HOST.  It is in form of tcp://<ip>:<port>
# DOCKER_CONFIG1  Directory where the docker config.json file for the Tenant 1, User 1
# DOCKER_CONFIG2  Directory where the docker config.json file for the Tenant 2, User 2
# DOCKER_CONFIG3  Directory where the docker config.json file for the Tenant 1, User 3
#
# flavorsDefault
#  default flavor 64 megabytes
#  medium flavor 128 megabytes
#  large flavor 258 megabytes
######################################################################################

load cli_helpers

@test "Check flavors" {
    #skip
	MEGABYTE=1048576
	DEFAULTFLAVOR=$((64 * $MEGABYTE)) # 64 megabytes
	MEDIUMFLAVOR=$((128 * $MEGABYTE)) # 128 megabytes
	LARGEFLAVOR=$((256 * $MEGABYTE)) # 256 megabytes
    # create daemons
	
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name useDefault busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{ .HostConfig.Memory }}' useDefault 
    [ "$status" -eq 0 ]
	[[ "$output" == "$DEFAULTFLAVOR" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name use64m -m 64m busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{ .HostConfig.Memory }}' use64m 
    [ "$status" -eq 0 ]
	[[ "$output" == "$DEFAULTFLAVOR" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name use128m -m 128m busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{ .HostConfig.Memory }}' use128m 
    [ "$status" -eq 0 ]
	[[ "$output" == "$MEDIUMFLAVOR" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name use256m -m 256m busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{ .HostConfig.Memory }}' use256m 
    [ "$status" -eq 0 ]
	[[ "$output" == "$LARGEFLAVOR" ]]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name use100m -m 100m busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect -f '{{ .HostConfig.Memory }}' use100m 
    [ "$status" -eq 0 ]
	[[ "$output" == "$DEFAULTFLAVOR" ]]




	
}





