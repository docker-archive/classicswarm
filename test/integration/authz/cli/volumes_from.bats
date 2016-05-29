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

@test "Check --volumes-from" {
	#skip
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create -v /data/db --name mongodbdata mongo:2.6 /bin/echo "Data-only container for mongodb."
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodbdata=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --volumes-from mongodbdata -p :27017 --name mongodb mongo:2.6 --smallfiles
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodb=$output
    # check that more than one container can link to same volume
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --volumes-from mongodbdata -p :27017 --name mongodb1 mongo:2.6 --smallfiles
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodb1=$output
    # check that container ids also work
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --volumes-from $cid_mongodbdata -p :27017 --name mongodb2 mongo:2.6 --smallfiles
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodb2=$output
    # check tenant isolation
    # ensure that tenant 2 can not access the volume from tenant 1
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --volumes-from mongodbdata -p :27017  mongo:2.6 --smallfiles
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --volumes-from $cid_mongodbdata -p :27017  mongo:2.6 --smallfiles
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
    # ensure that tenant 2 can create his own with same name
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 create -v /data/db --name mongodbdata mongo:2.6 /bin/echo "Data-only container for mongodb."
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    [ $cid_mongodbdata != $output ]
 
    run checkInvariant
    [ $status = 0 ]  
}






