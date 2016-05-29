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

@test "Check --volumes-from and link" {
    skip "work in progress"
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 create -v /data/db --name mongodbdata mongo:2.6 /bin/echo "Data-only container for mongodb."
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodbdata=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --volumes-from mongodbdata -p :27017 --name mongodb mongo:2.6 --smallfiles
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_mongodb=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --link mongodb  -p :1026 --name orion fiware/orion:latest -dbhost mongodb
    echo "$output"
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    cid_orion=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 port orion
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    array=( ${output} )
    echo ${array[2]}
    [ "$status" -eq 0 ]
    sleep 30
#   results=$(curl ${array[2]}/v1/contextTypes -s -S --header 'Content-Type: application/json' --header 'Accept: application/json')
#   echo $results
#   echo $output
#   [ "$status" -eq 0 ]
    sleep 30
    results=$((curl ${array[2]}/v1/contextEntities/Room1 -s -S --header 'Content-Type: application/json' --header 'Accept: application/json' -X POST -d @- | python -mjson.tool) <<EOF
{
    "attributes": [
        {
            "name": "temperature",
            "type": "float",
            "value": "23"
        },
        {
            "name": "pressure",
            "type": "integer",
            "value": "720"
        }
    ]
}
EOF)

   echo $results
   echo $output
#   goodresponse="\"code\": \"200\", \"reasonPhrase\": \"OK\"" 
#   goodresponse="code": "200", "reasonPhrase": "OK"" 
#   goodresponse='"code": "200", "reasonPhrase": "OK"' 
   goodresponse="\"200\"" 
   [[ $results == *"$goodresponse"* ]]
#   [[ $results == *'"statusCode": { "code": "200", "reasonPhrase": "OK" }'* ]]
#   [ "$status" -ne 0 ]

 
    run checkInvariant
    [ $status = 0 ]  
}

