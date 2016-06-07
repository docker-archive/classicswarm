#!/bin/bash

display_usage() {
	echo "This script is used to launch bats files to test multi-tenant swarm." 
	echo "Environment variables found in cli.properties must be set to reflect your configuration." 
	echo -e "\nUsage:\n$0 [options] \n"
	echo -e "\nexamples:\n"
	echo -e "\n$0 -H tcp://0.0.0.0:2376 -i  -f \"./cli\""
	echo -e "\n$0 -H tcp://0.0.0.0:2376 -i  -f \"./cli/run_inspect.bats ./cli/volumes_from.bats \""
	echo -e "\noptions:\n"
	echo -e "-h help\n"
	echo -e "-a authentication method. Valid options are None or Keystone. None is default. \n"
	echo -e "-H swarm host url. Defaults to environment variable DOCKER_HOST\n"
	echo -e "-f bats files and/or directories to run. Defaults to ./ \n"
	echo -e "-i enable invariant check.\n"
        exit 0
	}
AUTH="None"
INVARIANT=false
SWARM_HOST=$DOCKER_HOST
BAT_FILES="./"
while getopts ":hia:H:f:" opt; do
  case $opt in
    h)
      display_usage
      ;;
	a)
	  AUTH=${OPTARG}
	  ;;
	H)
      SWARM_HOST=${OPTARG}
	  ;;
	f)
      BAT_FILES=${OPTARG}
	  ;;
	i)
	  INVARIANT=true
	  ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      display_usage
      ;;
  esac
done

export DOCKER_CONFIG1=~/swarm_cli_test/docker-user-0
export DOCKER_CONFIG2=~/swarm_cli_test/docker-user-2
export DOCKER_CONFIG3=~/swarm_cli_test/docker-user-1
export TENANT_NAME_1=docker-tenant-0
export TENANT_NAME_2=docker-tenant-2
export USER_NAME_1=docker-user-0
export USER_NAME_2=docker-user-2
export USER_NAME_3=docker-user-1
export PASSWORD_1=secret
export PASSWORD_2=secret
export PASSWORD_3=secret
export KEYSTONE_IP=http://cloud.lab.fi-ware.org:4730

if [ -f cli.properties ]
then
. cli.properties
else
  echo "Info: cli.properties not found. Defaults will be used"
fi


if [ ! -d ${DOCKER_CONFIG1} ]
then
  echo "Info: make directory ${DOCKER_CONFIG1}"
  mkdir -p ${DOCKER_CONFIG1}
fi
if [ ! -d ${DOCKER_CONFIG2} ]
then
  echo "Info: make directory ${DOCKER_CONFIG2}"
  mkdir -p ${DOCKER_CONFIG2}
fi
if [ ! -d ${DOCKER_CONFIG3} ]
then
  echo "Info: make directory ${DOCKER_CONFIG3}"
  mkdir -p ${DOCKER_CONFIG3}
fi

if [ -z $SWARM_HOST ]
then
  echo "Info: SWARM_HOST is not set so will look for it on local host."
  port=$(netstat  -apn  2>/dev/null | egrep 'LISTEN.*swarm' | awk '{print $4}' | sed s/://g)
  if [ -z $port ]
  then
    echo "Error: swarm host can not be found."
	exit 1
  fi
  export SWARM_HOST=tcp://0.0.0.0:$port
fi

if [ $USER_NAME_1 == $USER_NAME_2 ] || [ $USER_NAME_2 == $USER_NAME_3 ] || [ $USER_NAME_1 == $USER_NAME_3 ]
then
  echo "Error: user names must all be different"
  exit 1
fi

if [ $AUTH == "Keystone" ]
then
  path_to_executable=$(which set_docker_conf.bash)
  if [ ! -x "$path_to_executable" ] ; then
    echo "Error: when authorization in Keystone, set_docker_conf.bash must be excutable in PATH"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG1} -t $TENANT_NAME_1 -u $USER_NAME_1 -p $PASSWORD_1 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "Error: set_docker_conf.bash failed when generating DOCKER_CONFIG1"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG2} -t $TENANT_NAME_2 -u $USER_NAME_2 -p $PASSWORD_2 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "Error: set_docker_conf.bash failed when generating DOCKER_CONFIG2"
    exit 1
  fi
  set_docker_conf.bash -d ${DOCKER_CONFIG3} -t $TENANT_NAME_1 -u $USER_NAME_3 -p $PASSWORD_3 -a $KEYSTONE_IP
  if [ $? -ne 0 ]
  then
    echo "Error: set_docker_conf.bash failed when generating DOCKER_CONFIG3"
    exit 1
  fi
else
  AUTH="None"
  rm -f ${DOCKER_CONFIG1}/config.json
  rm -f ${DOCKER_CONFIG2}/config.json
  rm -f ${DOCKER_CONFIG3}/config.json
  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_1\",\n\t\t\"X-Auth-Token\": \"user1_token\"\n\t}\n}" > "${DOCKER_CONFIG1}/config.json"
  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_2\",\n\t\t\"X-Auth-Token\": \"user2_token\"\n\t}\n}" > "${DOCKER_CONFIG2}/config.json"
  echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": \"$TENANT_NAME_1\",\n\t\t\"X-Auth-Token\": \"user3_token\"\n\t}\n}" > "${DOCKER_CONFIG3}/config.json"
fi

export SWARM_HOST=$SWARM_HOST
export DOCKER_ENGINE=$DOCKER_ENGINE
export DOCKER_CONFIG1=${DOCKER_CONFIG1}
export DOCKER_CONFIG2=${DOCKER_CONFIG2}
export DOCKER_CONFIG3=${DOCKER_CONFIG3}

echo "Info: DOCKER_CONFIG1 is ${DOCKER_CONFIG1}"
cat "${DOCKER_CONFIG1}/config.json"
echo "Info: DOCKER_CONFIG2 is ${DOCKER_CONFIG2}"
cat "${DOCKER_CONFIG2}/config.json"
echo "Info: DOCKER_CONFIG3: ${DOCKER_CONFIG3}"
cat "${DOCKER_CONFIG3}/config.json"


echo "Info: SWARM_HOST is $SWARM_HOST"
echo "Info: DOCKER_ENGINE is $DOCKER_ENGINE"
echo "Info: Authentication Method is $AUTH"
echo "Info: invariant check enabled: ${INVARIANT}"
echo "Info: bat files to run: $BAT_FILES"

start=$(date +'%s')
bats $BAT_FILES 
echo "Test elapse time: $(($(date +'%s') - $start)) seconds"
