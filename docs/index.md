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

## Release Notes

### Version 0.2.0 (April 16, 2015)

For complete information on this release, see the
[0.2.0 Milestone project page](https://github.com/docker/swarm/wiki/0.2.0-Milestone-Project-Page).
In addition to bug fixes and refinements, this release adds the following:

* Increased API coverage. Swarm now supports more of the Docker API. For
details, see
[the API README](https://github.com/docker/swarm/blob/master/api/README.md).

* Swarm Scheduler: Spread strategy. Swarm has a new default strategy for
ranking nodes, called "spread". With this strategy, Swarm will optimize
for the node with the fewest running containers. For details, see the
[scheduler strategy README](https://github.com/docker/swarm/blob/master/scheduler/strategy/README.md)

* Swarm Scheduler: Soft affinities. Soft affinities allow Swarm more flexibility
in applying rules and filters for node selection. If the scheduler can't obey a
filtering rule, it will discard the filter and fall back to the assigned
strategy. For details, see the [scheduler filter README](https://github.com/docker/swarm/tree/master/scheduler/filter#soft-affinitiesconstraints).

* Better integration with Compose. Swarm will now schedule inter-dependent
containers on the same host. For details, see
[PR #972](https://github.com/docker/compose/pull/972).

## Pre-requisites for running Swarm

You must install Docker 1.4.0 or later on all nodes. While each node's IP need not
be public, the Swarm manager must be able to access each node across the network.

To enable communication between the Swarm manager and the Swarm node agent on
each node, each node must listen to the same network interface (TCP port).
Follow the set up below to ensure you configure your nodes correctly for this
behavior.

> **Note**: Swarm is currently in beta, so things are likely to change. We
> don't recommend you use it in production yet.

## Install Swarm

The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

```bash
$ docker pull swarm
```

## Set up a secured Swarm cluster using docker-machine

`docker-machine` is a tool to provision docker nodes locally or within a Cloud provider infrastructure.

You can setup a Swarm cluster (**secured by default** using TLS) with this tool.

### Installation

See the steps in the [Docker Machine documentation](https://docs.docker.com/machine/) to install `docker-machine`

Make sure you have **Virtualbox** installed locally, this will be the driver used to create our Swarm Virtual Machines

### Create a token

First, create a Swarm token.  Optionally, you can use another discovery service.
See the Swarm documentation on alternative solutions in [Discovery Documentation](https://github.com/docker/swarm/blob/master/discovery/README.md)

To create the token, first create a Machine. This example will use VirtualBox.

```
$ docker-machine create -d virtualbox local
```

Load the Machine configuration into your shell:

```
$ eval "$(docker-machine env local)"
```

Then run generate the token using the Swarm Docker image:

```
$ docker run swarm create
1257e0f0bbb499b5cd04b4c9bdb2dab3
```
Once you have the token, you can create the cluster.

### Launch the Swarm manager

Use this command to launch the *Swarm Manager*:

```
docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-master \
    --swarm-discovery token://<TOKEN-FROM-ABOVE> \
    swarm-master
```

The *Swarm Manager* is the machine in charge of orchestrating and scheduling containers
on the entire cluster. The *Swarm Manager* rules a set of *Swarm-Agents* (also called **nodes** or **docker nodes**).

### Launch Swarm agents

Now that the *Swarm Manager* is up and running, we can launch as many *Swarm
Agents* as we want using:

```
docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery token://<TOKEN-FROM-ABOVE> \
    swarm-agent-00
```

We can create more: `swarm-agent-01`, `swarm-agent-02`, etc..

*Swarm agents* are responsible for hosting containers, they are regular docker daemons and
we communicate with them using the standard *docker remote API*.

### Point the docker cli to our Swarm Manager

Last step is to point to the machine running the *Swarm Manager* so we can use the `docker` command on it. To do so we are going to load those informations into our environment with:

```
eval "$(docker-machine env --swarm swarm-master)"
```

### Time to talk to our Swarm!

Now that the setup is done, we can use the `docker` command on our cluster:

```
$ docker info
Containers: 1
Nodes: 1
 swarm-master: 192.168.99.100:2376
  └ Containers: 2
  └ Reserved CPUs: 0 / 4
  └ Reserved Memory: 0 B / 999.9 MiB
Nodes: 2
 swarm-agent-00: 192.168.99.101:2376
  └ Containers: 1
  └ Reserved CPUs: 0 / 4
  └ Reserved Memory: 0 B / 999.9 MiB
Nodes: 3
 swarm-agent-01: 192.168.99.102:2376
  └ Containers: 1
  └ Reserved CPUs: 0 / 4
  └ Reserved Memory: 0 B / 999.9 MiB

$ docker ps
[...]
```


## Set up a Swarm cluster manually (insecure)

> **Warning**: use these steps only for debugging and testing purposes and if your network environment is secured (use of firewall, etc.)

Each Swarm node will run a Swarm node agent. The agent registers the referenced
Docker daemon, monitors it, and updates the discovery backend with the node's status.

The following example uses the Docker Hub based `token` discovery service:

1. Create a Swarm cluster using the `docker` command.

    ```bash
    $ docker run --rm swarm create
    6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>
    ```

    The `create` command returns a unique cluster ID (`cluster_id`). You'll need
    this ID when starting the Swarm agent on a node.

2. Log into **each node** and do the following.

    1. Start the Docker daemon with the `-H` flag. This ensures that the Docker remote API on *Swarm Agents* is available over TCP for the *Swarm Manager*.

        ```bash
    		$ docker -H tcp://0.0.0.0:2375 -d
        ```

    2. Register the Swarm agents to the discovery service. The node's IP must be accessible from the Swarm Manager. Use the following command and replace with the proper `node_ip` and `cluster_id` to start an agent:

        ```bash
        docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>
        ```

        For example:

        ```bash
        $ docker run -d swarm join --addr=172.31.40.100:2375 token://6856663cdefdec325839a4b7e1de38e8
        ```

3. Start the Swarm manager on any machine or your laptop. The following command
illustrates how to do this:

      ```bash
    	docker run -d -p <swarm_port>:2375 swarm manage token://<cluster_id>
      ```

4. Once the manager is running, check your configuration by running `docker info` as follows:

      ```bash
    	docker -H tcp://<manager_ip:manager_port> info
      ```

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

      ```bash
    	$ unset DOCKER_TLS_VERIFY
      ```

## Using the docker CLI

You can now use the regular Docker CLI to access your nodes:

```bash
docker -H tcp://<manager_ip:manager_port> info
docker -H tcp://<manager_ip:manager_port> run ...
docker -H tcp://<manager_ip:manager_port> ps
docker -H tcp://<manager_ip:manager_port> logs ...
```

## List nodes in your cluster

You can get a list of all your running nodes using the `swarm list` command:

```bash
docker run --rm swarm list token://<cluster_id>
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

```bash
swarm manage --tlsverify --tlscacert=<CACERT> --tlscert=<CERT> --tlskey=<KEY> [...]
```

Please refer to the [Docker documentation](https://docs.docker.com/articles/https/)
for more information on how to set up TLS authentication on Docker and generating
the certificates.

> **Note**: Swarm certificates must be generated with `extendedKeyUsage = clientAuth,serverAuth`.

## Discovery services

See the [Discovery service](https://docs.docker.com/swarm/discovery/) document
for more information.

## Advanced Scheduling

See [filters](https://docs.docker.com/swarm/scheduler/filter/) and [strategies](https://docs.docker.com/swarm/scheduler/strategy/) to learn
more about advanced scheduling.

## Swarm API

The [Docker Swarm API](https://docs.docker.com/swarm/API/) is compatible with
the [Docker remote
API](http://docs.docker.com/reference/api/docker_remote_api/), and extends it
with some new endpoints.

## Getting help

Docker Swarm is still in its infancy and under active development. If you need
help, would like to contribute, or simply want to talk about the project with
like-minded individuals, we have a number of open channels for communication.

* To report bugs or file feature requests: please use the [issue tracker on Github](https://github.com/docker/machine/issues).

* To talk about the project with people in real time: please join the `#docker-swarm` channel on IRC.

* To contribute code or documentation changes: please submit a [pull request on Github](https://github.com/docker/machine/pulls).

For more information and resources, please visit the [Getting Help project page](https://docs.docker.com/project/get-help/).
