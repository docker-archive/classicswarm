SwarmNode="http://127.0.0.1:2379"
#EngineIP="192.168.99.100"
#EngineNode="http://"$EngineIP":2376"
#echo "SwarmNode=$SwarmNode EngineNode=$EngineNode"
#echo "Count all containers + 1 ..."
#docker -H $EngineIP:2375 ps -a | wc -l
#Get user token
echo "Getting user token..."
UserToken=Space1
echo $UserToken
sleep 1
#echo "Asking Info with valid token..."
#curl -H "X-Auth-Token: $UserToken"  $SwarmNode/info
sleep 1
echo "Creating a container..."
#curl -v --data-binary @redis1.json -H "X-Auth-Token:$UserToken" -H "Content-type: application/json" $SwarmNode/containers/create?name=Doron
ContainerId=$(curl --data-binary @redis1.json -H "X-Auth-Token:$UserToken" -H "Content-type: application/json" $SwarmNode/containers/create | jq -r '.Id')
echo $ContainerId
sleep 1
echo "Starting the container..."
curl -v -X POST -H "X-Auth-Token:$UserToken" $SwarmNode/v1.18/containers/$ContainerId/start

echo "Testing short ID useage..."
prefix=${ContainerId:0:4}
echo $prefix
curl -H "X-Auth-Token:$UserToken" "$SwarmNode/containers/$prefix/top" | jq '.'

#curl -v -X POST -H "X-Auth-Token:$UserToken" $SwarmNode/v1.18/containers/123/start

sleep 1
echo "Listing containers..."
curl -H "X-Auth-Token: $UserToken"  $SwarmNode/containers/json?all=1 | jq '.'
sleep 1

#echo "showing the underline lable mechanisem..."
#docker -H 127.0.0.1:2376 inspect $ContainerId
#sleep 1

echo "Getting Another user token..."
UserToken2=Space2
echo $UserToken2
#curl -v --data-binary @redis1.json -H "X-Auth-Token:$UserToken2" -H "Content-type: application/json" $SwarmNode/containers/create?name=Doron
echo "Listing containers with the other user token..."
curl -H "X-Auth-Token: $UserToken2"  $SwarmNode/containers/json?all=1 | jq '.'
sleep 1

#echo "Listing containers with the admin user token..."
#curl -H "X-Auth-Token: admin"  $SwarmNode/containers/json?all=1 | jq '.'
#sleep 1

echo "Inspecting the container..."
curl -H "X-Auth-Token:$UserToken" $SwarmNode/containers/$ContainerId/json | jq '.'
sleep 2

echo "Getting the container logs..."
curl -H "X-Auth-Token:$UserToken" "$SwarmNode/containers/$ContainerId/logs?stderr=1&stdout=1&timestamps=1&follow=0&tail=10"
sleep 2

echo "Stopping the container..."
curl -X POST -H "X-Auth-Token:$UserToken" $SwarmNode/containers/$ContainerId/stop
sleep 1

echo "Inspecting the container again..."
curl -H "X-Auth-Token:$UserToken" $SwarmNode/containers/$ContainerId/json | jq '.'
sleep 1

echo "Trying to Delete the container with the other user token..."
curl -v -X DELETE -H "X-Auth-Token:$UserToken2" $SwarmNode/containers/$ContainerId
sleep 1

echo "Deleting the container..."
curl -X DELETE -H "X-Auth-Token:$UserToken" $SwarmNode/containers/$ContainerId
sleep 1

echo "Creating container with name..."
curl -v --data-binary @redis1.json -H "X-Auth-Token:$UserToken" -H "Content-type: application/json" $SwarmNode/containers/create?name=Doron
echo "Inspecting container with name..."
curl -H "X-Auth-Token:$UserToken" $SwarmNode/containers/Doron/json | jq '.'
echo "Done name tests - run again for duplicate name tests or delete"

echo "Tesing Ping"
curl -H "X-Auth-Token:$UserToken" $SwarmNode/_ping

echo "Testing image"
curl -X POST $SwarmNode/images/create?fromImage=ubuntu

#echo "Listing containers..."
#curl -H "X-Auth-Token: $UserToken"  $SwarmNode/containers/json?all=1 | jq '.'
#echo "Count all containers + 1 ..."
#docker -H $EngineIP:2376 ps -a | wc -l

