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
Dokku, Compose, Krane, Deis, DockerUI, Shipyard, Drone, Jenkins... and,
of course, the Docker client itself.

Like other Docker projects, Swarm follows the "batteries included but removable"
principle. It ships with a simple scheduling backend out of the box, and as
initial development settles, an API will develop to enable pluggable backends.
The goal is to provide a smooth out-of-box experience for simple use cases, and
allow swapping in more powerful backends, like Mesos, for large scale production
deployments.

## Pre-requisites for running Swarm

You must install Docker 1.4.0 or later on all nodes. While each node's IP need not
be public, the Swarm manager must be able to access each node across the network.

To enable communication between the Swarm manager and the Swarm node agent on each 
node, each node must listen to the same network interface (tcp port). Follow the set
up below to ensure you configure your nodes correctly for this behavior.

> **Note**: Swarm is currently in beta, so things are likely to change. We
> don't recommend you use it in production yet.

## Install Swarm

The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

```bash
docker pull swarm
```

## Set up Swarm nodes

Each swarm node will run a swarm node agent. The agent registers the referenced
Docker daemon, monitors it, and updates the discovery backend with the node's status.

The following example uses the Docker Hub based `token` discovery service:

1. Create a Swarm cluster using the `docker` command.

    ```bash
    $ docker run --rm swarm create
    6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>
    ```

    The create command returns a unique cluster id (`cluster_id`). You'll need
    this id when starting the Swarm agent on a node.

2. Log into **each node** and do the following.

    1. Start the docker daemon with the `-H` flag. This ensures that the docker remote API on *Swarm Agents* is available over TCP for the *Swarm Manager*.

    		$ docker -H tcp://0.0.0.0:2375 -d

    2. Register the Swarm agents to the discovery service. The node's IP must be accessible from the Swarm Manager. Use the following command and replace with the proper `node_ip` and `cluster_id` to start an agent:

        	docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>

        For example:

        ```bash
        $ docker run -d swarm join --addr=172.31.40.100:2375 token://6856663cdefdec325839a4b7e1de38e8
        ```

3. Start the Swarm manager on any machine or your laptop. The following command
illustrates how to do this:

    	docker run -d -p <swarm_port>:2375 swarm manage token://<cluster_id>

4. Once the manager is running, check your configuration by running `docker info` as follows:

    	docker -H tcp://<manager_ip:manager_port> info
    
    For example, if you run the manager locally on your machine:

    ```bash
    $ docker -H tcp://0.0.0.0:2375 info
    Containers: 0
    Nodes: 3
     agent-2: 172.31.40.102:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 514.5 MiB
     agent-1: 172.31.40.101:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 514.5 MiB
     agent-0: 172.31.40.100:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 514.5 MiB
    ```
    
    If you are running a test cluster without TLS enabled, you may get an error. In that case, be sure to unset `DOCKER_TLS_VERIFY` with:
    
    	$ unset DOCKER_TLS_VERIFY

## Using the docker CLI

You can now use the regular `docker` CLI to access your nodes:

```
docker -H tcp://<manager_ip:manager_port> info
docker -H tcp://<manager_ip:manager_port> run ...
docker -H tcp://<manager_ip:manager_port> ps
docker -H tcp://<manager_ip:manager_port> logs ...
```

## List nodes in your cluster

You can get a list of all your running nodes using the `swarm list` command:

```
`docker run --rm swarm list token://<cluster_id>`
<node_ip:2375>
```

For example:

```bash
$ docker run --rm swarm list token://6856663cdefdec325839a4b7e1de38e8
172.31.40.100:2375
172.31.40.101:2375
172.31.40.102:2375
```

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

See the [Discovery service](discovery.md) document for more information.

## Advanced Scheduling

See [filters](scheduler/filter.md) and [strategies](scheduler/strategy.md) to learn
more about advanced scheduling.

## Swarm API

The [Docker Swarm API](API.md) is compatible with the [Docker
remote API](http://docs.docker.com/reference/api/docker_remote_api/), and extends it
with some new endpoints.
