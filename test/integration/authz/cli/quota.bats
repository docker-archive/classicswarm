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

#300MB quota per tenant


load cli_helpers
@test "single tenant, single user, basic memory quota limitation" {
		run setup
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 250M --name container2 busybox
        [ "$status" -ne 0 ]   
        run setup 
}
@test "single tenant, multiple users, basic memory quota limitation" {
		run setup
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 40M --name container2 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -ne 0 ] 
        run setup      
}
@test "multi tenant, multiple users, basic memory quota limitation" {
		run setup
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -ne 0 ]
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -ne 0 ]
        run setup
}

@test "single tenant, single user, quota gets bigger after removing container" {
        run setup
		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]

		#trying to create container with over limited quota
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -ne 0 ]

		#stop container
		#run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST stop container1
        #[ "$status" -eq 0 ]
        #removing container
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST rm -f container1
        [ "$status" -eq 0 ]

		#now create container will have enough available quota
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -eq 0 ]
        run setup
}

@test "single tenant, multi users, quota gets bigger after removing container" {
        run setup
		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]

		#trying to create container with over limited quota
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -ne 0 ]

		#stop container
		#run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST stop container1
        #[ "$status" -eq 0 ]
        #removing container
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST rm -f container1
        [ "$status" -eq 0 ]

		#now create container will have enough available quota
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -eq 0 ]
        run setup
}

@test "multi tenant, multi users, quota gets bigger after removing container" {
        run setup
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -ne 0 ]

		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 50M --name container1 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -ne 0 ]
                
		#stop container
		#run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST stop container1
        #[ "$status" -eq 0 ]
        #removing container 
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST rm -f container1
        [ "$status" -eq 0 ]
        
		#stop container
		#run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST stop container1
        #[ "$status" -eq 0 ]
        #removing container 
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST rm -f container1
        [ "$status" -eq 0 ]

		#now create container will have enough available quota
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -eq 0 ]
	
		#now create container will have enough available quota
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 260M --name container2 busybox
        [ "$status" -eq 0 ]	
        run setup
}


@test "single tenant, single user, stop and start container doesn't change quota " {
        run setup
		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 150M --name container1 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST stop container1
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST start container1
        [ "$status" -eq 0 ]

		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 15M --name container3 busybox
        [ "$status" -ne 0 ]	   
        run setup 
}
@test "multi tenant, multi user, stop and start container doesn't change quota " {
		run setup
		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 150M --name container1 busybox
        [ "$status" -eq 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 150M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 50M --name container2 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST stop container1
        [ "$status" -eq 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST stop container1
        [ "$status" -eq 0 ]

		#can't create container with the existing name on the same tenant
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 40M --name container2 busybox
        [ "$status" -ne 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]

        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST start container1
        [ "$status" -eq 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST start container1
        [ "$status" -eq 0 ]

		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -ne 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 15M --name container3 busybox
        [ "$status" -ne 0 ]	
        run setup
}

@test "multi tenant, multi user, final quota test " {
		run setup
		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 150M --name container1 busybox
        [ "$status" -eq 0 ]
		run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 150M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 50M --name container2 busybox
        [ "$status" -eq 0 ]
		run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -ne 0 ]
		run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST rm -f container1
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 150M --name container3 busybox
        [ "$status" -eq 0 ]
        run setup
}

@test "single tenant, single user, test delete container directly through DOCKER_ENGINE (not through swarm) " {
        skip
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker -H $DOCKER_ENGINE rm -f container1$TENANT_NAME_1
        [ "$status" -eq 0 ]

		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 200M --name container1 busybox
        [ "$status" -eq 0 ]
}

@test "single tenant, multi user, test delete container directly through DOCKER_ENGINE (not through swarm) " {
        skip
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]
        run docker -H $DOCKER_ENGINE rm -f container2$TENANT_NAME_1
        [ "$status" -eq 0 ]

		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 100M --name container2 busybox
        [ "$status" -eq 0 ]
}
@test "single tenant, multi user, test delete container directly through DOCKER_ENGINE (not through swarm) " {
        skip
        run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG3 -H $SWARM_HOST run -tid -m 140M --name container2 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 140M --name container1 busybox
        [ "$status" -eq 0 ]
        
        run docker -H $DOCKER_ENGINE rm -f container2$TENANT_NAME_1
        [ "$status" -eq 0 ]
        run docker -H $DOCKER_ENGINE rm -f container1$TENANT_NAME_2
        [ "$status" -eq 0 ]

		run docker --config $DOCKER_CONFIG1 -H $SWARM_HOST run -tid -m 100M --name container2 busybox
        [ "$status" -eq 0 ]
        run docker --config $DOCKER_CONFIG2 -H $SWARM_HOST run -tid -m 300M --name container1 busybox
        [ "$status" -eq 0 ]

		run checkInvariant
	    [ $status = 0 ]
}

