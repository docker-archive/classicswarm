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
CMD_UNSUPPORTED="Error response from daemon: Command Not Supported!"

@test "commit unsupported" {
    #skip
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create --name top busybox top  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	topConfig1Id=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 commit $topConfig1Id test/busybox_top:v1
	[ "$status" -ne 0 ]
	[[ "$output" == *"$CMD_UNSUPPORTED"* ]]
	
}

@test "login unsupported" {
    #skip
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 login -e user@gmail.com -u user -p secret server
	[ "$status" -ne 0 ]
	[[ "$output" == *"$CMD_UNSUPPORTED"* ]]
	
}

@test "info unsupported disable by user" {
    #skip Requires export SWARM_APIFILTER_FILE=./test/integration/authz/data/apitfilter.json
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 info
	[ "$status" -ne 0 ]
	[[ "$output" == *"$CMD_UNSUPPORTED"* ]]
	
}

@test "top unsupported disable by user" {
    #skip Requires export SWARM_APIFILTER_FILE=./test/integration/authz/data/apitfilter.json
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 top acontainer_name
	[ "$status" -ne 0 ]
	[[ "$output" == *"$CMD_UNSUPPORTED"* ]]
	
}



