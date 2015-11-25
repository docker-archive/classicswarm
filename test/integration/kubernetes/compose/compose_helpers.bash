#!/bin/bash

load ../kubernetes_helpers

function docker-compose_swarm() {
	 DOCKER_HOST=${SWARM_HOSTS[0]} docker-compose "$@"
}
