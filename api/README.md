---
page_title: Docker Swarm API
page_description: Swarm API
page_keywords: docker, swarm, clustering, api
---

# Docker Swarm API

The Docker Swarm API is mostly compatible with the [Docker Remote API](https://docs.docker.com/reference/api/docker_remote_api/). This document is an overview of the differences between the Swarm API and the Docker Remote API.

## Missing endpoints

Some endpoints have not yet been implemented and will return a 404 error.

```
GET "/images/get"
GET "/containers/{name:.*}/attach/ws"

POST "/build"
POST "/images/load"
POST "/images/{name:.*}/push"
POST "/images/{name:.*}/tag"
```

## Endpoints which behave differently

* `GET "/containers/{name:.*}/json"`: New field `Node` added:

```json
"Node": {
	"Id": "ODAI:IC6Q:MSBL:TPB5:HIEE:6IKC:VCAM:QRNH:PRGX:ERZT:OK46:PMFX",
	"Ip": "0.0.0.0",
	"Addr": "http://0.0.0.0:4243",
	"Name": "vagrant-ubuntu-saucy-64",
    },
```
* `GET "/containers/{name:.*}/json"`: `HostIP` replaced by the the actual Node's IP if `HostIP` is `0.0.0.0`

* `GET "/containers/json"`: Node's name prepended to the container name.

* `GET "/containers/json"`: `HostIP` replaced by the the actual Node's IP if `HostIP` is `0.0.0.0`

* `GET "/containers/json"` : Containers started from the `swarm` official image are hidden by default, use `all=1` to display them.

* `GET "/images/json"` : Use '--filter node=\<Node name\>' to show images of the specific node.

## Docker Swarm documentation index

- [User guide](https://docs.docker.com/swarm/)
- [Discovery options](https://docs.docker.com/swarm/discovery/)
- [Sheduler strategies](https://docs.docker.com/swarm/scheduler/strategy/)
- [Sheduler filters](https://docs.docker.com/swarm/scheduler/filter/)
