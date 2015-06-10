#discovery-stage.hub.docker.com

Docker Swarm comes with a simple discovery service built into the [Docker Hub](http://hub.docker.com)

The discovery service is still in alpha stage and currently hosted at `https://discovery-stage.hub.docker.com`

#####Create a new **authenticated** cluster

`-> POST https://discovery-stage.hub.docker.com/v2/clusters/<username>`

`<- 401 Unauthorize {Www-Authenticate: Bearer realm=https://auth.docker.io/token,service=swarm-discovery}`

`-> GET https://auth.docker.io/token?service=swarm-discovery {Authorization: Basic <base64(user:pass)>}`

`<- {"token": "<authToken>"}`

`-> POST https://discovery-stage.hub.docker.com/v2/clusters/<username> {Authorization: Bearer <authToken>}`

`<- <token>`


####Create a new **anonymous** cluster

`-> POST https://discovery-stage.hub.docker.com/v2/clusters`

`<- <token>`


#####Add new nodes to a cluster
`-> PUT https://discovery-stage.hub.docker.com/v2/clusters/<token> Request body: "<ip>:<port1>"`

`<- OK`

`-> PUT https://discovery-stage.hub.docker.com/v2/clusters/<token> Request body: "<ip>:<port2>")`

`<- OK`


#####List nodes in a cluster
`-> GET https://discovery-stage.hub.docker.com/v2/clusters/<token>`

`<- ["<ip>:<port1>", "<ip>:<port2>"]`


#####Delete a **authenticated** cluster (all the nodes in a cluster)
`-> DELETE https://discovery-stage.hub.docker.com/v2/clusters/<token>`

`<- 401 Unauthorize {Www-Authenticate: Bearer realm=https://auth.docker.io/token,service=swarm-discovery}`

`-> GET https://auth.docker.io/token?service=swarm-discovery {Authorization: Basic <base64(user:pass)>}`

`<- {"token": "<authToken>"}`

`-> DELETE https://discovery-stage.hub.docker.com/v2/clusters/<token> {Authorization: Bearer <authToken>}`

`<- OK`


#####Delete an **anonymous** cluster (all the nodes in a cluster)
`-> DELETE https://discovery-stage.hub.docker.com/v2/clusters/<token>`

`<- OK`
