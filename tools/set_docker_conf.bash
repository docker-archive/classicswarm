#!/bin/bash

KEYSTONE_IP=${KEYSTONE_IP:-http://cloud.lab.fi-ware.org:4730}
CONFIG_DIRECTORY=${DOCKER_CONF:-~/.docker}

verbose=false

print_env() {
    echo '---------------------------'
    echo 'CONFIG_DIRECTORY:	'$CONFIG_DIRECTORY
    echo 'TENANT_NAME:		'$TENANT_NAME
    echo 'USERNAME:		'$USERNAME
    echo 'PASSWORD:		'$PASSWORD
    echo 'KEYSTONE_IP:		'$KEYSTONE_IP
    echo '---------------------------'
}

display_usage() { 
    echo -e "\nThis script updates docker config file with Keystone"
	echo -e "tenant/token variables Keystone server IP must be specified"
    echo -e "either as script input or added to environment as KEYSTONE_IP"
    echo -e "variable. The rest (USERNAME, PASSWORD...etc.) the script"
    echo -e "may get from environment, so in most cases it's enough to"
    echo -e "source OpenStack openrc file"
    echo -e "\nIn case environment missing those variables those must be supplied as script arguments"
    echo -e "If no arguments specified will try to use defaults below:"
    print_env
    echo -e "\nUsage:\n$0 [-d CONFIG_DIRECTORY] [-t TENANT_NAME] [-u USER_NAME] [-p PASSWORD] [-a KEYSTONE_IP] [-v|-verbose] [-h|-help]\n"
    echo -e  "\nExample:\n$0 -d ~/.docker -t \"my cloud\" -u myfiwareuser -p myfiwarepassword -a cloud.lab.fi-ware.org:4730 \n"
} 

validate_env() {
    [[ $TENANT_NAME && $USERNAME && $PASSWORD && $KEYSTONE_IP ]] || { print_env; echo -e 'ERROR! Missing one or more required variables.\nUse -h to get help and usage information.\n'; exit 1; }
}

get_tenants() {
	out=`curl -s -X GET ${KEYSTONE_IP}tenants -H "X-Auth-Token: $test" -H "Content-Type: application/json"|python -m json.tool`
	
	tenants=`echo $out | python -c "import json,sys;obj=json.load(sys.stdin);print [str(xxx['name']) for xxx in obj['tenants']]" | tr -d "[]'"`
    echo "$tenants"	
	#echo "$tenants" | sed -r 's/(\[|\])//g'
}

while getopts ":hhelp:f:d:t:u:p:a:vverbose" opt; do
      case $opt in
          h|help )
                display_usage >&2
                exit 1
                ;;
          d)
                CONFIG_DIRECTORY=${OPTARG}
                ;;                
          t)
                TENANT_NAME=$OPTARG
                ;;
          u)
                USERNAME=$OPTARG
                ;;
          p)
                PASSWORD=$OPTARG
                ;;
          a)
                KEYSTONE_IP=$OPTARG
                ;;
          v)
                verbose=true
                ;;
          \?)
                echo "Invalid option: -$OPTARG" >&2
                exit 1
                ;;
          :)
                echo "Option -$OPTARG requires an argument." >&2
                exit 1
                ;;
      esac
done

mkdir -p $CONFIG_DIRECTORY
DOCKER_CONF="${CONFIG_DIRECTORY}/config.json"
validate_env

[[ $KEYSTONE_IP != http://* ]] && KEYSTONE_IP=http://$KEYSTONE_IP
[[ $KEYSTONE_IP != http://*:* ]] && KEYSTONE_IP=$KEYSTONE_IP:5000
[[ $KEYSTONE_IP != http://*:*/ ]] && KEYSTONE_IP=$KEYSTONE_IP/
[[ $KEYSTONE_IP != http://*:*/*/ ]] && KEYSTONE_IP=${KEYSTONE_IP}v2.0/


$verbose && echo -e '\n---------------------------'
$verbose && echo 'Using following environment'
$verbose && print_env

out=`curl -s -X POST ${KEYSTONE_IP}tokens -H "Content-Type: application/json" -d '{"auth": {"tenantName": "'"$TENANT_NAME"'", "passwordCredentials":{"username": "'"$USERNAME"'", "password": "'"$PASSWORD"'"}}}'| python -m json.tool|grep id|tail -3|head -2|awk -F"\"id\":" '{print $1,$2}'|awk -F"," '{print $1,$2}'`

$verbose && echo "running curl -s -X POST ${KEYSTONE_IP}tokens -H \"Content-Type: application/json\" -d '{\"auth\": {\"tenantName\": \"'\"$TENANT_NAME\"'\", \"passwordCredentials\":{\"username\": \"'\"$USERNAME\"'\", \"password\": \"'\"$PASSWORD\"'\"}}}'| python -m json.tool|grep id|tail -3|head -2|awk -F\"\"id\":\" '{print $1,$2}'|awk -F\",\" '{print $1,$2}'"
$verbose && echo $out
test=( $out )
token=${test[0]}
tenant=${test[1]}

$verbose && echo -e "\nTOKEN: $token"
$verbose && echo -e "TENANT: $tenant\n"

#validate returned token
[ "$token" ] || { echo $out; $verbose && print_env; echo -e 'ERROR Getting token from keystone!\n'; exit 1; }

test=`echo $token|tr -d '\"'`
out=`curl -s -X GET ${KEYSTONE_IP}tenants -H "X-Auth-Token: $test" -H "Content-Type: application/json"|grep ${tenant}`
$verbose && echo "Validating token using: curl -s -X GET ${KEYSTONE_IP}tenants -H \"X-Auth-Token: $test\" -H \"Content-Type: application/json\""
[ "$out" ] || { echo $out; $verbose && print_env; echo -e "ERROR Validating token ${token}!\n"; exit 1; }

if [ -f ${DOCKER_CONF} ]; then
	$verbose && echo "File ${DOCKER_CONF} exist"
	sed -i '/\,/! s/.*\"X-Auth-TenantId\".*/            "X-Auth-TenantId": '${tenant}'/' $DOCKER_CONF
	sed -i '/\,/ s/.*\"X-Auth-TenantId\".*/            "X-Auth-TenantId": '${tenant}',/' $DOCKER_CONF

	sed -i '/\,/! s/.*\"X-Auth-Token\".*/            "X-Auth-Token": '${token}'/' $DOCKER_CONF
	sed -i '/\,/ s/.*\"X-Auth-Token\".*/            "X-Auth-Token": '${token}',/' $DOCKER_CONF
else
	$verbose && echo "File ${DOCKER_CONF} not exist"
	echo -e "{\n\t\"HttpHeaders\": {\n\t\t\"X-Auth-TenantId\": ${tenant},\n\t\t\"X-Auth-Token\": ${token}\n\t}\n}" > $DOCKER_CONF
fi

$verbose && echo -e '\n\n---------------------------'
$verbose && echo "New config file: ${DOCKER_CONF}"
$verbose && echo -e '---------------------------\n'
$verbose && cat $DOCKER_CONF

[ "$token" ]
#exit 0
