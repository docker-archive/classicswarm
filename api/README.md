---
page_title: Docker Swarm API
page_description: Swarm API
page_keywords: docker, swarm, clustering, api
---

# Docker Swarm API

The Docker Swarm API is compatible with the [Offical Docker API](https://docs.docker.com/reference/api/docker_remote_api/):

Here are the main differences:

## Some endpoints are not (yet) implemented

```
GET "/images/get"
GET "/images/{name:.*}/get"
GET "/containers/{name:.*}/attach/ws"

POST "/commit"
POST "/build"
POST "/images/create"
POST "/images/load"
POST "/images/{name:.*}/push"
POST "/images/{name:.*}/tag"

DELETE "/images/{name:.*}"
```

## Some endpoints have more information

* `GET "/containers/{name:.*}/json"`: New field `Node` added:

```json
"Node": {
        "ID": "ODAI:IC6Q:MSBL:TPB5:HIEE:6IKC:VCAM:QRNH:PRGX:ERZT:OK46:PMFX",
	"IP": "0.0.0.0",
	"Addr": "http://0.0.0.0:4243",
	"Name": "vagrant-ubuntu-saucy-64",
	"Cpus": 1,
	"Memory": 2099654656,
	"Labels": {
            "executiondriver": "native-0.2",
            "kernelversion": "3.11.0-15-generic",
            "operatingsystem": "Ubuntu 13.10",
            "storagedriver": "aufs"
	    }
    },
```
* `GET "/containers/{name:.*}/json"`: `HostIP` replaced by the the actual Node's IP if `HostIP` is `0.0.0.0`

* `GET "/containers/json"`: Node's name prepended to the container name.

* `GET "/containers/json"`: `HostIP` replaced by the the actual Node's IP if `HostIP` is `0.0.0.0`

* `GET "/containers/json"` : Containers started from the `swarm` official image are hidden by default, use `all=1` to display them.

