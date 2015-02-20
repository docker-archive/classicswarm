---
page_title: Docker Swarm
page_description: Swarm: a Docker-native clustering system
page_keywords: docker, swarm, clustering
---

# Docker Swarm

Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual host.

Swarm serves the standard Docker API, so any tool which already communicates
with a Docker daemon can use Swarm to transparently scale to multiple hosts:
Dokku, Compose, Krane, Flynn, Deis, DockerUI, Shipyard, Drone, Jenkins... and,
of course, the Docker client itself.

Like other Docker projects, Swarm follows the "batteries included but removable"
principle. It ships with a simple scheduling backend out of the box, and as
initial development settles, an API will develop to enable pluggable backends.
The goal is to provide a smooth out-of-box experience for simple use cases, and
allow swapping in more powerful backends, like Mesos, for large scale production
deployments.

## Installation

> **Note**: The only requirement for Swarm nodes is they all run the _same_ release
> Docker daemon (version `1.4.0` and later), configured to listen to a `tcp`
> port that the Swarm manager can access.

The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

```bash
docker pull swarm
```

## Nodes setup

Each swarm node will run a swarm node agent which will register the referenced
Docker daemon, and will then monitor it, updating the discovery backend to its
status.

The following example uses the Docker Hub based `token` discovery service:

```bash
# create a cluster
$ docker run --rm swarm create
6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>

# on each of your nodes, start the swarm agent
#  <node_ip> doesn't have to be public (eg. 192.168.0.X),
#  as long as the swarm manager can access it.
$ docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>

# start the manager on any machine or your laptop
$ docker run -t -p <swarm_port>:2375 -t swarm manage token://<cluster_id>

# use the regular docker cli
$ docker -H tcp://<swarm_ip:swarm_port> info
$ docker -H tcp://<swarm_ip:swarm_port> run ...
$ docker -H tcp://<swarm_ip:swarm_port> ps
$ docker -H tcp://<swarm_ip:swarm_port> logs ...
...

# list nodes in your cluster
$ docker run --rm swarm list token://<cluster_id>
<node_ip:2375>
```

> **Note**: In order for the Swarm manager to be able to communicate with the node agent on
each node, they must listen to a common network interface. This can be achieved
by starting with the `-H` flag (e.g. `-H tcp://0.0.0.0:2375`).


## TLS

Swarm supports TLS authentication between the CLI and Swarm but also between
Swarm and the Docker nodes. _However_, all the Docker daemon certificates and client
certificates **must** be signed using the same CA-certificate.

In order to enable TLS for both client and server, the same command line options
as Docker can be specified:

`swarm manage --tlsverify --tlscacert=<CACERT> --tlscert=<CERT> --tlskey=<KEY> [...]`

Please refer to the [Docker documentation](https://docs.docker.com/articles/https/)
for more information on how to set up TLS authentication on Docker and generating
the certificates.

> **Note**: Swarm certificates must be generated with`extendedKeyUsage = clientAuth,serverAuth`.

## Discovery services

See the [Discovery service](../discovery) document for more information.

## Advanced Scheduling

See [filters](../scheduler/filter) and [strategies](../scheduler/strategy) to learn
more about advanced scheduling.
