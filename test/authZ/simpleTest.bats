#!/usr/bin/env bats

setup() {
	SwarmNode="http://127.0.0.1:2379"
	echo "Getting user token..."
	UserToken=Space1
	UserToken2=Space2
	ContainerId=$(curl --data-binary @redis1.json -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" -H "Content-type: application/json" $SwarmNode/containers/create | jq -r '.Id')
	curl -v -X POST -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" $SwarmNode/v1.18/containers/$ContainerId/start
}

@test "Testing usage with ID prefix..." {
	prefix=${ContainerId:0:4}
	HTTPCode=$(curl -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" --write-out %{http_code} --silent --output /dev/null "$SwarmNode/containers/$prefix/top")
	[ "$HTTPCode" -eq 200 ]
}

@test "Inspect a container - Verifying running state..." {
	Status=$(curl -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" $SwarmNode/containers/$ContainerId/json | jq -r '.State.Status')
	echo $Status
	[ "$Status" == "running" ]
}

@test "Tesing multi tenancy on logs..." {
	 HTTPCode=$(curl -H "X-Auth-Token:$UserToken2" -H "X-Auth-TenantId:$UserToken2" --write-out %{http_code} --silent --output /dev/null "$SwarmNode/containers/$ContainerId/logs?stderr=1&stdout=1&timestamps=1&follow=0&tail=10")
	[ "$HTTPCode" -eq 401 ]
}

@test "Tesing multi tenancy on Checking list containers..." {
        HTTPCode=$(curl -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" --write-out %{http_code} --silent --output /dev/null  "$SwarmNode/containers/json?all=1")
        [ "$HTTPCode" -eq 200 ]
}


@test "Tesing multi tenancy on list containers (empty tenant)..." {
	Response=(curl -H "X-Auth-Token:Space2" -H "X-Auth-TenantId:Space2" "http://127.0.0.1:2379/containers/json?all=1")
	[ ${#Response} -eq 4 ]
}

@test "Testing useage of auto generated names..." {
	Name=$(curl -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" $SwarmNode/containers/$ContainerId/json | jq -r '.Name')
	Status=$(curl -H "X-Auth-Token:$UserToken" -H "X-Auth-TenantId:$UserToken" $SwarmNode/containers$Name/json | jq -r '.State.Status')
        echo $Status
        [ "$Status" == "running" ]

}

}



