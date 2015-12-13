<!--[metadata]>
+++
title = "Docker Swarm API"
description = "Swarm API"
keywords = ["docker, swarm, clustering,  api"]
aliases = ["/swarm/api/"]
[menu.main]
parent="smn_swarm_ref"
+++
<![end-metadata]-->

# Docker Swarm API

The Docker Swarm API is mostly compatible with the [Docker Remote API](https://docs.docker.com/reference/api/docker_remote_api/). This document is an overview of the differences between the Swarm API and the Docker Remote API.

## Missing endpoints

Some endpoints have not yet been implemented and will return a 404 error.

```
POST "/images/create" : "docker import" flow not implement
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

* `POST "/containers/create"`: `CpuShares` in `HostConfig` sets the number of CPU cores allocated to the container.

## Docker Swarm documentation index

- [Docker Swarm overview](https://docs.docker.com/swarm/)
- [Discovery options](https://docs.docker.com/swarm/discovery/)
- [Scheduler strategies](https://docs.docker.com/swarm/scheduler/strategy/)
- [Scheduler filters](https://docs.docker.com/swarm/scheduler/filter/)
