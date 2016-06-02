#!/bin/bash

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
    run rmall $DOCKER_CONFIG1
    run rmall $DOCKER_CONFIG2
	run rmall_volumes $DOCKER_CONFIG1
	run rmall_volumes $DOCKER_CONFIG2

}

teardown()  {
    :
}

checkInvariant() {
 if [ "${INVARIANT}" != true ] ; then
  return 0
 fi 	
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
notAuthorized() {
	NOTAUTHORIZED="Error response from daemon: Not Authorized!"
#	return 0  #temporary fix to all tests to run until we support all the container commands
#	run docker -H $SWARM_HOST --config $1 attach $2	
#   if !( [ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]] ); then
#       return 200
#   fi
#   if !([ "$status" -ne 0 ] ); then
#        return 1
#    fi
#    if !([[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#        return 100
#    fi
#	run docker -H $SWARM_HOST --config $1 exec $2 ls
#    if !( [ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]] ); then
#        return 2
#    fi
#	run docker -H $SWARM_HOST --config $1 export $2
#    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#        return 3
#    fi
	run docker -H $SWARM_HOST --config $1 inspect $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 4
    fi
	run docker -H $SWARM_HOST --config $1 kill $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 5
    fi
	run docker -H $SWARM_HOST --config $1 logs $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 8
    fi
	run docker -H $SWARM_HOST --config $1 port $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 9
    fi
#	run docker -H $SWARM_HOST --config $1 rename $2 "foo"
#    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#        return 10
#    fi
#	run docker -H $SWARM_HOST --config $1 restart $2
#   if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#       return 11
#   fi
	run docker -H $SWARM_HOST --config $1 rm $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 12
    fi
	run docker -H $SWARM_HOST --config $1 start $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 13
    fi
#	run docker -H $SWARM_HOST --config $1 stats $2
#   if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#        return 14
#   fi
#    run docker -H $SWARM_HOST --config $1 top $2
#    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#        return 15
#    fi
	run docker -H $SWARM_HOST --config $1 unpause $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 16
    fi
	run docker -H $SWARM_HOST --config $1 update -m 4m $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 17
    fi
#    run docker -H $SWARM_HOST --config $1 wait $2
#    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
#       return 18
#    fi
	run docker -H $SWARM_HOST --config $1 stop $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 19
    fi
	run docker -H $SWARM_HOST --config $1 pause $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 20
    fi
	run docker -H $SWARM_HOST --config $1 pause $2
    if !([ "$status" -ne 0 ] && [[ "$output" == *"$NOTAUTHORIZED"* ]]); then
        return 21
    fi
	#'
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
     run docker -H $SWARM_HOST --config $1 rm -fv $i
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






