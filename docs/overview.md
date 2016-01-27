<!--[metadata]>
+++
title = "Swarm Overview"
description = "Swarm: a Docker-native clustering system"
keywords = ["docker, swarm,  clustering"]
[menu.main]
parent="workw_swarm"
weight=-99
+++
<![end-metadata]-->

# Docker Swarm overview

Docker Swarm is native clustering for Docker. You can use it to create a single virtual Docker host from a pool of Docker hosts.

To scale up and run more containers, add more nodes to your Swarm cluster. In production, a Swarm cluster can have 1,000 nodes running 50,000 containers with no performance degradation.

Docker tools are compatible with Swarm. Because the Docker Swarm API is based on the standard Docker Remote API, tools that communicate with a Docker host work with Docker Swarm. These tools include Dokku, Docker Compose, Krane, Flynn, Deis, DockerUI, Shipyard, Drone, Jenkins, the Docker client, and others.

Like other Docker projects, Docker Swarm follows the "swap, plug, and play"
principle by supporting pluggable backends. For example, on a large-scale Swarm cluster, you can use Apache Mesos to do scheduling instead of Swarm's built-in tools.

## Understand Swarm creation

There are different ways to create a Swarm cluster. This is a simple high-level
description of how to do it.

On a Docker Engine host, use the `docker run swarm create` command to get a token from the discovery backend. When you create Swarm managers and nodes later on, you use this token to authenticate each one as a member of your cluster.

Then use the `docker swarm create` command to create a Swarm manager and multiple Swarm nodes. For high availability, create an odd number of managers, and put each one on a dedicated host.

When you create managers and nodes using the discovery token, each one registers with the discovery backend as a member of your Swarm cluster. The discovery service keeps a current list of each cluster member, and shares this information with the managers.

Also, when you create Swarm managers and nodes on different hosts, you enable and configure the TCP ports so the managers can communicate directly with each node.

To manage the Swarm cluster, connect to the Swarm manager using your Docker Client or any other software that works with the Docker API. Then, start running containers on your Swarm like you would on any Docker Engine.

## Docker Machine
You can use Docker Machine to set up a Docker Swarm on your computer, cloud service provider, or data center.

If you're running Windows or Mac OS X, get Docker Machine and VirtualBox by installing Docker Toolbox. Then, you can use Machine to build and explore a Docker Swarm on your computer. This is the best way to [get started with Swarm for the first time](install-w-machine.md).

If you're interested, you can also [manually install Swarm and contribute to the Swarm project](install-manual.md).

## Discovery services

To keep track of the managers and nodes in a cluster, Docker Swarm needs a discovery backend. By default, Swarm uses the free tier of Docker Hub's discovery service. ***Only use this tier for low volume purposes. For high-volume production clusters, you must choose and set up one of [these discovery service backends](discovery.md).***

## Advanced scheduling

When the Swarm gets your command to run a container, it assigns the container to one of the nodes and runs it there. By default, the Swarm uses the `spread` strategy, which sends containers to the node that has the most CPU and RAM *available* at that moment. For more control over scheduling, see the
[strategies](scheduler/strategy.md) and [filters](scheduler/filter.md)
topics.

## Swarm API

The [Docker Swarm API](api/swarm-api.md) is compatible with
the [Docker Remote
API](http://docs.docker.com/reference/api/docker_remote_api/) and extends it
with some new endpoints.

# Getting help

Docker Swarm is under active development. If you need help, would like to
contribute, or simply want to talk about the project with like-minded
individuals, we have some open channels for communication.

* To report bugs or file feature requests: please use the [issue tracker on Github](https://github.com/docker/swarm/issues).

* To talk about the project with people in real time: please join the `#docker-swarm` channel on IRC.

* To contribute code or documentation changes: please submit a [pull request on Github](https://github.com/docker/swarm/pulls).

For more information and resources, please visit the [Getting Help project page](https://docs.docker.com/project/get-help/).
