#!/bin/bash

display_usage() { 
	echo "This script is used to launch cli.bats to test multi-tenant swarm." 
	echo "Environment variables found in cli.properties must be set to reflect your configuration." 
	echo -e "\nUsage:\n$0 [-h] [-a <None | Keystone> ]\n"
	echo -e "-h help\n"
	echo -e "-a authentication method. None is default. Keystone Authentication.\n"
        exit 0 
	}
AUTH="None"
while getopts ":ha:" opt; do
  case $opt in
    h)
      display_usage
      ;;
	a)
	  AUTH=${OPTARG}
	  ;; 
    \?)
      echo "Invalid option: -$OPTARG" >&2
      display_usage
      ;;
  esac
done
  
. cli.properties


if [ ! -d ${DOCKER_CONFIG1} ]
then
  echo "DOCKER_CONFIG1 is not directory"
  exit 1 
fi
if [ ! -d ${DOCKER_CONFIG2} ]
then
  echo "DOCKER_CONFIG2 is not directory"
  exit 1 
fi
if [ ! -d ${DOCKER_CONFIG3} ]
then
  echo "DOCKER_CONFIG3 is not directory"
  exit 1 
fi

if [ -z $SWARM_HOST ]
then
  echo "SWARM_HOST is not set"
  exit 1 
fi

if [ $USER_NAME_1 == $USER_NAME_2 ] || [ $USER_NAME_2 == $USER_NAME_3 ] || [ $USER_NAME_1 == $USER_NAME_3 ]
then
  echo "user names must all be different"
  exit 1
fi

if [ $AUTH == "Keystone" ]
then
  path_to_executable=$(which set_docker_conf.bash) 
  if [ ! -x "$path_to_executable" ] ; then
    echo "set_docker_conf.bash must be excutable in PATH"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG1} -t $TENANT_NAME_1 -u $USER_NAME_1 -p $PASSWORD_1 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "set_docker_conf.bash failed when generating DOCKER_CONFIG1"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG2} -t $TENANT_NAME_2 -u $USER_NAME_2 -p $PASSWORD_2 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "set_docker_conf.bash failed when generating DOCKER_CONFIG2"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG3} -t $TENANT_NAME_1 -u $USER_NAME_3 -p $PASSWORD_3 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "set_docker_conf.bash failed when generating DOCKER_CONFIG3"
    exit 1
  fi
else
  AUTH="None"
  rm ${DOCKER_CONFIG1}/config.json
  rm ${DOCKER_CONFIG2}/config.json
  rm ${DOCKER_CONFIG3}/config.json

  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_1\",\n\t\t\"X-Auth-Token\": \"NA\"\n\t}\n}" > "${DOCKER_CONFIG1}/config.json"
  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_2\",\n\t\t\"X-Auth-Token\": \"NA\"\n\t}\n}" > "${DOCKER_CONFIG2}/config.json"
  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_1\",\n\t\t\"X-Auth-Token\": \"NA\"\n\t}\n}" > "${DOCKER_CONFIG3}/config.json"
fi

export SWARM_HOST=$SWARM_HOST
export DOCKER_CONFIG1=${DOCKER_CONFIG1}
export DOCKER_CONFIG2=${DOCKER_CONFIG2}
export DOCKER_CONFIG3=${DOCKER_CONFIG3}

echo "DOCKER_CONFIG1: ${DOCKER_CONFIG1}"
cat "${DOCKER_CONFIG1}/config.json"
echo "DOCKER_CONFIG2: ${DOCKER_CONFIG2}"
cat "${DOCKER_CONFIG2}/config.json"
echo "DOCKER_CONFIG3: ${DOCKER_CONFIG3}"
cat "${DOCKER_CONFIG3}/config.json"


echo "SWARM_HOST: $SWARM_HOST"
echo "Authentication Method: $AUTH"
echo "run cli.bats"
bats cli.bats 
