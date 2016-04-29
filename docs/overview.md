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

Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual Docker host. Because Docker Swarm serves the standard
Docker API, any tool that already communicates with a Docker daemon can use
Swarm to transparently scale to multiple hosts. Supported tools include, but
are not limited to, the following:

- Dokku
- Docker Compose
- Krane
- Jenkins

And of course, the Docker client itself is also supported.

Like other Docker projects, Docker Swarm follows the "swap, plug, and play"
principle. As initial development settles, an API will develop to enable
pluggable backends.  This means you can swap out the scheduling backend
Docker Swarm uses out-of-the-box with a backend you prefer. Swarm's swappable design provides a smooth out-of-box experience for most use cases, and allows large-scale production deployments to swap for more powerful backends, like Mesos.

## Understand Swarm cluster creation

The first step to creating a Swarm cluster on your network is to pull the Docker Swarm image. Then, using Docker, you configure the Swarm manager and all the nodes to run Docker Swarm. This method requires that you:

* open a TCP port on each node for communication with the Swarm manager
* install Docker on each node
* create and manage TLS certificates to secure your cluster

As a starting point, the manual method is best suited for experienced
administrators or programmers contributing to Docker Swarm. The alternative is
to use `docker-machine` to install a cluster.

Using Docker Machine, you can quickly install a Docker Swarm on cloud providers
or inside your own data center. If you have VirtualBox installed on your local
machine, you can quickly build and explore Docker Swarm in your local
environment. This method automatically generates a certificate to secure your
cluster.

Using Docker Machine is the best method for users getting started with Swarm for the first time. To try the recommended method of getting started, see [Get Started with Docker Swarm](install-w-machine.md).

If you are interested manually installing or interested in contributing, see [Build a Swarm cluster for production](install-manual.md).

## Discovery services

To dynamically configure and manage the services in your containers, you use a discovery backend with Docker Swarm. For information on which backends are available, see the [Discovery service](discovery.md) documentation.

## Advanced Scheduling

To learn more about advanced scheduling, see the
[strategies](scheduler/strategy.md) and [filters](scheduler/filter.md)
documents.

## Swarm API

The [Docker Swarm API](swarm-api.md) is compatible with
the [Docker remote
API](http://docs.docker.com/reference/api/docker_remote_api/), and extends it
with some new endpoints.

## Getting help

Docker Swarm is still in its infancy and under active development. If you need
help, would like to contribute, or simply want to talk about the project with
like-minded individuals, we have a number of open channels for communication.

* To report bugs or file feature requests: please use the [issue tracker on Github](https://github.com/docker/swarm/issues).

* To talk about the project with people in real time: please join the `#docker-swarm` channel on IRC.

* To contribute code or documentation changes: please submit a [pull request on Github](https://github.com/docker/swarm/pulls).

For more information and resources, please visit the [Getting Help project page](https://docs.docker.com/project/get-help/).
