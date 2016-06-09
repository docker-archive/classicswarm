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

The Docker Swarm API is mostly compatible with the [Docker Remote
API](https://docs.docker.com/engine/reference/api/docker_remote_api/). This
document is an overview of the differences between the Swarm API and the Docker
Remote API.

## Missing endpoints

Some endpoints have not yet been implemented and will return a 404 error.

```
POST "/images/create" : "docker import" flow not implement
```

## Endpoints which behave differently

<table>
    <tr>
        <th>Endpoint</th>
        <th>Differences</th>
    </tr>
    <tr>
        <td>
            <code>GET "/containers/{name:.*}/json"</code>
        </td>
        <td>
            New field <code>Node</code> added:<br />
<pre>
"Node": {
    "Id": "ODAI:IC6Q:MSBL:TPB5:HIEE:6IKC:VCAM:QRNH:PRGX:ERZT:OK46:PMFX",
    "Ip": "0.0.0.0",
    "Addr": "http://0.0.0.0:4243",
    "Name": "vagrant-ubuntu-saucy-64"
}
</pre>
        </td>
    </tr>
    <tr>
        <td>
            <code>GET "/containers/{name:.*}/json"</code>
        </td>
        <td>
            <code>HostIP</code> replaced by the the actual Node's IP if <code>HostIP</code> is <code>0.0.0.0</code>
        </td>
    </tr>
    <tr>
        <td>
            <code>GET "/containers/json"</code>
        </td>
        <td>
            Node's name prepended to the container name.
        </td>
    </tr>
    <tr>
        <td>
            <code>GET "/containers/json"</code>
        </td>
        <td>
            <code>HostIP</code> replaced by the the actual Node's IP if <code>HostIP</code> is <code>0.0.0.0</code>
        </td>
    </tr>
    <tr>
        <td>
            <code>GET "/containers/json"</code>
        </td>
        <td>
            Containers started from the <code>swarm</code> official image are hidden by default, use <code>all=1</code> to display them.
        </td>
    </tr>
    <tr>
        <td>
            <code>GET "/images/json"</code>
        </td>
        <td>
            Use <code>--filter node=&lt;Node name&gt;</code> to show images of the specific node.
        </td>
    </tr>
    <tr>
        <td>
            <code>POST "/containers/create"</code>
        </td>
        <td>
            <code>CpuShares</code> in <code>HostConfig</code> sets the number of CPU cores allocated to the container.
        </td>
    </tr>
</table>

## Registry Authentication

During container create calls, the Swarm API will optionally accept an `X-Registry-Auth` header.
If provided, this header is passed down to the engine if the image must be pulled
to complete the create operation.

The following two examples demonstrate how to utilize this using the existing Docker CLI

### Authenticate using registry tokens

> **Note:** This example requires Docker Engine 1.10 with auth token support.
> For older Engine versions, refer to [authenticate using username and
> password](#authenticate-using-username-and-password)

This example uses the [`jq` command-line utility](https://stedolan.github.io/jq/).
To run this example, install `jq` using your package manager (`apt-get install jq` or `yum install jq`).

```bash
REPO=yourrepo/yourimage
REPO_USER=yourusername
read -s PASSWORD
AUTH_URL=https://auth.docker.io/token

# obtain a JSON token, and extract the "token" value using 'jq'
TOKEN=$(curl -s -u "${REPO_USER}:${PASSWORD}" "${AUTH_URL}?scope=repository:${REPO}:pull&service=registry.docker.io" | jq -r ".token")
HEADER=$(echo "{\"registrytoken\":\"${TOKEN}\"}"|base64 -w 0 )
echo HEADER=$HEADER
```

Add the header you've calculated to your `~/.docker/config.json`:

```json
"HttpHeaders": {
    "X-Registry-Auth": "<HEADER string from above>"
}
```

You can now authenticate to the registry, and run private images on Swarm:

```bash
$ docker run --rm -it yourprivateimage:latest
```


Be aware that tokens are short-lived and will expire quickly.


### Authenticate using username and password

> **Note:** this authentication method stores your credentials unencrypted
> on the filesystem. Refer to [Authenticate using registry tokens](#authenticate-using-registry-tokens)
> for a more secure approach.


First, calculate the header

```bash
REPO_USER=yourusername
read -s PASSWORD
HEADER=$(echo "{\"username\":\"${REPO_USER}\",\"password\":\"${PASSWORD}\"}" | base64 -w 0 )
unset PASSWORD
echo HEADER=$HEADER
```

Add the header you've calculated to your `~/.docker/config.json`:

```json
"HttpHeaders": {
    "X-Registry-Auth": "<HEADER string from above>"
}
```

You can now authenticate to the registry, and run private images on Swarm:

```bash
$ docker run --rm -it yourprivateimage:latest
```



## Docker Swarm documentation index

- [Docker Swarm overview](https://docs.docker.com/swarm/)
- [Discovery options](https://docs.docker.com/swarm/discovery/)
- [Scheduler strategies](https://docs.docker.com/swarm/scheduler/strategy/)
- [Scheduler filters](https://docs.docker.com/swarm/scheduler/filter/)
