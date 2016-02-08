#!/usr/bin/env bats

load authz_helpers
load ../../../tools/set_docker_conf

function teardown() {
	stop_docker
	swarm_manage_cleanup
}

@test "basic memory quota limitation" {
	start_docker 1
	swarm_manage_multy_tenant
	run docker_swarm run -d -m 50M --name container1 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container2 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]
}

@test "single tenant, memory quota limitation" {
	start_docker 1
	swarm_manage_multy_tenant
	run docker_swarm run -d -m 50M --name container1 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container2 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]

	docker_swarm rm container1
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -eq 0 ]

	run docker_swarm run -d -m 50M --name container4 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]
}


@test "simple memory quota limitation with keystone" {
	start_docker 1
	swarm_manage_multy_tenant
	loginToKeystoneTenant1
	run docker_swarm run -d -m 50M --name container1 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container2 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]
}


@test "two tenants, memory quota limitation" {
	start_docker 1
	swarm_manage_multy_tenant

	#First tenant
	loginToKeystoneTenant1

	run docker_swarm run -d -m 50M --name container1 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container2 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]

	#Second tenant
	loginToKeystoneTenant2
	run docker_swarm run -d -m 50M --name container3 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container4 hello-world
	[ "$status" -eq 0 ]
	run docker_swarm run -d -m 50M --name container5 hello-world
	[ "$status" -ne 0 ]
	[[ "${lines[0]}" == *"Tenant memory quota limit reached"* ]]
}

@test "wrong keystone credentials" {
	temp=$TENANT_NAME
	export TENANT_NAME='wrong tenantname'

	run loginToKeystoneTenant1
	[ "$status" -ne 0 ]
	export TENANT_NAME=$temp
	run loginToKeystoneTenant1
	[ "$status" -eq 0 ]

	start_docker 1
	swarm_manage_multy_tenant

	run docker_swarm run -d -m 50M --name container1 hello-world
	[ "$status" -eq 0 ]
}