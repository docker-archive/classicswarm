# Libswarm openstack backend

*libswarm* is a toolkit for composing network services.

It defines a standard interface for services in a distributed system to communicate with each other. This lets you:

1. Compose complex architectures from reusable building blocks
2. Avoid vendor lock-in by swapping any service out with another

openstack backend allows you to use standard docker APIs on openstack backend.
I installed Openstack with nova docker driver from StackFroge (https://github.com/stackforge/nova-docker/).

## Installation

First get the go dependencies:

```sh
go get github.com/docker/libswarm/...
go get github.com/goinggo/mapstructure
```

Then you can compile `swarmd` with:

```sh
go install github.com/docker/libswarm/swarmd
```

If `$GOPATH/bin` is in your `PATH`, you can invoke `swarmd` from the CLI.
 
## Built-in services

### Docker server

*Maintainer: Ben Firshman*

This service runs a Docker remote API server, allowing the Docker client and
other Docker tools to control libswarm services. With no arguments, it listens
on port `4243`, but you can specify any port you like using `tcp://0.0.0.0:9999`,
`unix:///tmp/docker` etc.

### Openstack
* Shripad J Nadgowda *

## Testing libswarm with swarmd and openstack backend

Run swarmd without arguments to list available services:

```
./swarmd
```

To launch swarmd with openstack backend, first setup environment variables for openstack access

```
#cat creds
export OS_TENANT_NAME=admin
export OS_USERNAME=admin
export OS_PASSWORD=admin
export OS_AUTH_URL="http://9.xxx.xxx.xx:5000/v2.0/"

#source creds
```

Then you can launch openstack backend service :

```
./swarmd 'dockerserver tcp://localhost:4245' 'openstack --flavor=m1.tiny'
```

For every container instance spawn with this backend, it will have same flavor. 
Onlt at the launch time you can specify flavor from all available flavors from openstack.

Currently, openstack backend service supports ONLY following docker APIs:

```
a) Create new container: POST /containers/create 
b) List containers:      GET /containers/json
c) Stop Container:       POST /containers/<id>/stop
d) Stop Container:       POST /containers/<id>/start
e) Delete Container:     DELETE /containers/<id>
```

### Example
a) Create Container
```sh
curl -i -H "Content-Type: application/json" -H "Accept: application/json" -X POST http://localhost:4245/containers/create -d '{"name":"demo1", "image":"cirros"}'
HTTP/1.1 201 Created
Content-Type: application/json
Date: Mon, 29 Sep 2014 05:04:06 GMT
Content-Length: 46

{"Id":"5e257004-df64-451f-b9a3-00d8aed61bc1"}
``` 

b) List containers
```sh
curl -H "Content-Type: application/json" -H "Accept: application/json" -X GET http://localhost:4245/containers/json | python -m json.tool
[
    {
        "Command": "",
        "Created": 1408775487,
        "Id": "5e257004-df64-451f-b9a3-00d8aed61bc1",
        "Image": "fe4c5130-0bd1-43a2-8455-6f15623a30d3",
        "Names": [
            "demo1"
        ],
        "Ports": null,
        "Status": "Up"
    },
    {
        "Command": "",
        "Created": 1408251739,
        "Id": "2e148ffb-4f6b-4550-84d7-e2c9df2d8738",
        "Image": "fe4c5130-0bd1-43a2-8455-6f15623a30d3",
        "Names": [
            "demo2"
        ],
        "Ports": null,
        "Status": "Up"
    }
]
``` 

c) Stop container
```sh
curl -i -H "ContendataType: application/json" -H "Accept: application/json" -X POST http://localhost:4245/containers/5e257004-df64-451f-b9a3-00d8aed61bc1/stop
HTTP/1.1 204 No Content
Date: Mon, 29 Sep 2014 05:19:42 GMT
``` 

d) Start container
```sh
curl -i -H "ContendataType: application/json" -H "Accept: application/json" -X POST http://localhost:4245/containers/5e257004-df64-451f-b9a3-00d8aed61bc1/start
HTTP/1.1 204 No Content
Date: Mon, 29 Sep 2014 05:19:34 GMT
``` 

b) Delete container
```sh
curl -i -H "ContendataType: application/json" -H "Accept: application/json" -X DELETE http://localhost:4245/containers/5e257004-df64-451f-b9a3-00d8aed61bc1
HTTP/1.1 204 No Content
Date: Mon, 29 Sep 2014 05:24:24 GMT
``` 