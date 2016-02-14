<!--[metadata]>
+++
aliases = ["api/swarm-api/", "/swarm/api/"]
title = "Docker Swarm API"
description = "Swarm API"
keywords = ["docker, swarm, clustering,  api"]
[menu.main]
parent="workw_swarm"
weight=99
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

# Registry Authentication

During container create calls, the swarm API will optionally accept a X-Registry-Config header.
If provided, this header will be passed down to the engine if the image must be pulled
to complete the create operation.

The following two examples demonstrate how to utilize this using the existing docker CLI

* CLI usage example using username/password:

    ```bash
# Calculate the header
REPO_USER=yourusername
read -s PASSWORD
HEADER=$(echo "{\"username\":\"${REPO_USER}\",\"password\":\"${PASSWORD}\"}"|base64 -w 0 )
unset PASSWORD
echo HEADER=$HEADER

# Then add the following to your ~/.docker/config.json
"HttpHeaders": {
    "X-Registry-Auth": "<HEADER string from above>"
}

# Now run a private image against swarm:
docker run --rm -it yourprivateimage:latest
```

* CLI usage example using registry tokens: (Requires engine 1.10 with new auth token support)

    ```bash
REPO=yourrepo/yourimage
REPO_USER=yourusername
read -s PASSWORD
AUTH_URL=https://auth.docker.io/token
TOKEN=$(curl -s -u "${REPO_USER}:${PASSWORD}" "${AUTH_URL}?scope=repository:${REPO}:pull&service=registry.docker.io" |
    jq -r ".token")
HEADER=$(echo "{\"registrytoken\":\"${TOKEN}\"}"|base64 -w 0 )
echo HEADER=$HEADER

# Update the docker config as above, but the token will expire quickly...
```


## Docker Swarm documentation index

- [Docker Swarm overview](https://docs.docker.com/swarm/)
- [Discovery options](https://docs.docker.com/swarm/discovery/)
- [Scheduler strategies](https://docs.docker.com/swarm/scheduler/strategy/)
- [Scheduler filters](https://docs.docker.com/swarm/scheduler/filter/)
