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
  

setup()  {
    run stopall $DOCKER_CONFIG1
    run rmall $DOCKER_CONFIG1
    run stopall $DOCKER_CONFIG2
    run rmall $DOCKER_CONFIG2
	run rmall_volumes $DOCKER_CONFIG1
	run rmall_volumes $DOCKER_CONFIG2

}

teardown()  {
    :
}

checkInvariant() {
 run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 ps -aq
 config1Array=(${lines[@]})
 run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 ps -aq
 config2Array=(${lines[@]})
 run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 ps -aq
 config3Array=(${lines[@]})
 for i in "${config1Array[@]}"
 do
     for j in "${config2Array[@]}" ; do
         if [ "${i}" == "${j}" ]; then
            echo " checkInvariant failed: container ${i} in both DOCKER_CONFIG1 and DOCKER_CONFIG2 "
            return 1
         fi
     done
  done
 if [ "${#config1Array[@]}" != "${#config3Array[@]}" ]; then
     echo " checkInvariant failed: DOCKER_CONFIG1 and DOCKER_CONFIG3 not identical on size ${#config1Array[@]} != ${#config3Array[@]} "
     return 1
 fi
 for (( i=0; i<"${#config1Array[@]}"; i++)) ; do
     if [ "${#config1Array[$i]}" != "${#config3Array[$i]}" ]; then
        echo " checkInvariant failed: DOCKER_CONFIG1 and DOCKER_CONFIG3 not identical on $i ${config1Array[$i]} != ${config3Array[$i]} "
        return 1
     fi
 done  
 return 0
}

stopall() {
   echo "stop all $1"
   run docker -H $SWARM_HOST --config $1 ps -q
   [ "$status" -eq 0 ]
   for i in "${lines[@]}"
   do
     run docker -H $SWARM_HOST --config $1 stop $i
     [ "$status" -eq 0 ]
   done
}

rmall() {
   echo "rm all $1"
   run docker -H $SWARM_HOST --config $1 ps -aq
   [ "$status" -eq 0 ]
   for i in "${lines[@]}"
   do
     run docker -H $SWARM_HOST --config $1 rm -v $i
     [ "$status" -eq 0 ]
   done
}

rmall_volumes() {
   echo "volume rm all $1"
   run docker -H $SWARM_HOST --config $1 volume ls -q
   for i in "${lines[@]}"
   do
     #v=echo $i | cut -d'/' -f2
     run docker -H $SWARM_HOST --config $1 volume rm $i
   done
}

@test "Check ps empty list" {
    skip "unskip when you want to check that all $DOCKER_CONFIGS are authorized before starting"    
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 ps
    echo $output
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    [ "${#lines[@]}" = 1 ]
    array=( ${lines[0]} )
    [ "${array[0]}" = "CONTAINER" ]
    [ "${array[1]}" = "ID" ]
    [ "${#array[@]}" = 8 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 ps
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    [ "${#lines[@]}" = 1 ]
    array=( ${lines[0]} )
    [ "${array[0]}" = "CONTAINER" ]
    [ "${array[1]}" = "ID" ]
    [ "${#array[@]}" = 8 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 ps
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    [ "${#lines[@]}" = 1 ]
    array=( ${lines[0]} )
    [ "${array[0]}" = "CONTAINER" ]
    [ "${array[1]}" = "ID" ]
    [ "${#array[@]}" = 8 ]

    run checkInvariant
    echo $output
    [ $status = 0 ]
}
@test "Check run and inspect" {
    #skip
	# run non daemons
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1  run --name  busy1 busybox
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect busy1
    [ "$status" -eq 0 ]   
    # same name different tenant
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2  run --name  busy1 busybox
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect busy1
    [ "$status" -eq 0 ]
    # different user of tenant
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3  run --name  busy2 busybox
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect busy2
    [ "$status" -eq 0 ]
    # tenant 2 trying to access tenant 1 container
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect busy2
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]

    # run daemons
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -d --name loop1 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect loop1
    [ "$status" -eq 0 ]
    inpectBusy1Config1=$output
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect loop1
    [ "$status" -eq 0 ]
    [ "$inpectBusy1Config1" = "$output" ]  
  
 
    # same name different tenant 
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -d --name loop1 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect loop1
    [ "$status" -eq 0 ]
    [ "$inpectBusy1Config1" != "$output" ]   

    # different user of tenant
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 run -d --name loop2 ubuntu /bin/sh -c "while true; do echo Hello world; sleep 1; done"  
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 inspect loop2
    [ "$status" -eq 0 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 inspect loop2
    [ "$status" -eq 0 ]
    run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 inspect loop2
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]

    run checkInvariant
    echo $output
    [ $status = 0 ]
}
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
    echo $output
    [ $status = 0 ]  
}

@test "Check volume management" {

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume create --name t1volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t1volume" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $(echo ${lines[0]} | cut -d'/' -f2) == "t1volume" ]]
	hostPlusVolumeNamet1=${lines[0]} 
	for v in "${lines[@]}"
    do	
	 [[ $v == *"t1volume" ]]
     run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume inspect $v
     [ "$status" -eq 0 ]
    done

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume inspect $hostPlusVolumeNamet1
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]

	
	# check that members of same tenant can assess volume
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 volume inspect ${hostPlusVolumeNamet1}
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG3 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ ${lines[0]}==${hostPlusVolumeNamet1} ]]

	# check isolation
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume create --name t2volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t2volume" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	[[ $(echo ${lines[0]} | cut -d'/' -f2) == "t2volume" ]]
	hostPlusVolumeNamet2=${lines[0]}
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume inspect ${hostPlusVolumeNamet2}
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume inspect ${hostPlusVolumeNamet2}
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	#[ "${#lines[@]}" = 1 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume rm ${hostPlusVolumeNamet2}
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]

	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume inspect ${hostPlusVolumeNamet1}
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]]
	
	# allowing same name
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume create --name t1volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t1volume" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	#[ "${#lines[@]}" = 2 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume inspect ${hostPlusVolumeNamet1}
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume rm t1volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t1volume" ]]	
	
	# without name
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume create 
    [ "$status" -eq 0 ]
	newvolume=$output
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume ls -q
    [ "$status" -eq 0 ]
    [[ "$output" != *"Error"* ]]
	#[ "${#lines[@]}" = 2 ]
	#run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume inspect $newvolume
    #[ "$status" -eq 0 ]
    #[[ "$output" != *"Error"* ]]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume rm $newvolume
    [ "$status" -eq 0 ]
    [[ "$output" == "$newvolume" ]]

	# remove volumes
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume rm t1volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t1volume" ]]	
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume rm t2volume
    [ "$status" -eq 0 ]
    [[ "$output" == "t2volume" ]]	

}
@test "Check volume binding" {
	#skip
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume create --name myvolume
    [ "$status" -eq 0 ]
    [[ "$output" == "myvolume" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -v myvolume:/data --name con1  busybox sh -c "echo tenant1 hello  > /data/file.txt"
    [ "$status" -eq 0 ]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume create --name myvolume
    [ "$status" -eq 0 ]
    [[ "$output" == "myvolume" ]]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -v myvolume:/data --name con2 busybox sh -c "echo tenant2 hello  > /data/file.txt"
    [ "$status" -eq 0 ]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -v myvolume:/data  --name con3  busybox sh -c "cat /data/file.txt"
	[ "$status" -eq 0 ]
	[[ "$output" == "tenant1 hello" ]]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 run -v myvolume:/data --name con4 busybox sh -c "cat /data/file.txt"
	[ "$status" -eq 0 ]
	[[ "$output" == "tenant2 hello" ]]	

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 rm con1
	[ "$status" -eq 0 ]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 rm con2
	[ "$status" -eq 0 ]
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 rm con3
	[ "$status" -eq 0 ]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 rm con4
	[ "$status" -eq 0 ]

	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 volume rm myvolume
	[ "$status" -eq 0 ]
	[[ "$output" == "myvolume" ]]	
	
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG2 volume rm myvolume
	[ "$status" -eq 0 ]
	[[ "$output" == "myvolume" ]]	



	
	# implicit volume without volume create 
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -v implicit_myvolume:/data busybox sh -c "echo tenant1 hello  > /data/file.txt"
    [ "$status" -eq 0 ]
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -v implicit_myvolume:/data busybox sh -c "cat /data/file.txt"
	[ "$status" -eq 0 ]
	[[ "$output" == "tenant1 hello" ]]
	
	# error case trying volume mount to host file system not permitted
	run docker -H $SWARM_HOST --config $DOCKER_CONFIG1 run -v /tmp:/data busybox sh -c "echo tenant1 hello  > /data/file.txt"
    [ "$status" -ne 0 ]
	[[ "$output" == *"Error"* ]]

}

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
    echo $output
    [ $status = 0 ]  
}



