<!--[metadata]>
+++
title = "Provision with Machine"
description = "Provision with Machine"
keywords = ["docker, Swarm, clustering, provision, Machine"]
[menu.main]
parent="workw_swarm"
weight=5
+++
<![end-metadata]-->


# Provision a Swarm cluster with Docker Machine

You can use Docker Machine to provision a Docker Swarm cluster. Machine is the
Docker provisioning tool. Machine provisions the hosts, installs Docker Engine
on them, and then configures the Docker CLI client. With Machine's Swarm
options, you can also quickly configure a Swarm cluster as part of this
provisioning.

This page explains the commands you need to provision a basic Swarm cluster on a
local Mac or Windows computer using Machine. Once you understand the process,
you can use it to setup a Swarm cluster on a cloud provider, or inside your
company's data center.

If this is the first time you are creating a Swarm cluster, you should first
learn about Swarm and its requirements by [installing a Swarm for
evaluation](install-w-machine.md) or [installing a Swarm for
production](install-manual.md). If this is the first time you have used Machine,
you should take some time to [understand Machine before
continuing](https://docs.docker.com/machine).


## What you need

If you are using Mac OS X or Windows and have installed with Docker Toolbox, you
should already have Machine installed. If you need to install, see the
instructions for [Mac OS X](https://docs.docker.com/engine/installation/mac/) or
[Windows](https://docs.docker.com/engine/installation/mac/).

Machine supports installing on AWS, Digital Ocean, Google Cloud Platform, IBM
Softlayer, Microsoft Azure and Hyper-V, OpenStack, Rackspace, VirtualBox, VMware
Fusion&reg;, vCloud&reg; Air<sup>TM</sup> and vSphere&reg;.  In this example,
you'll use VirtualBox to run several VMs based on the `boot2docker.iso` image.
This image is a small-footprint Linux distribution for running Engine.

The Toolbox installation gives you VirtualBox and the `boot2docker.iso` image
you need. It also gives you the ability provision on all the systems Machine
supports.

**Note**:These examples assume you are using Mac OS X or Windows, if you like you can also [install Docker Machine directly on a Linux
system](http://docs.docker.com/machine/install-machine).

## Provision a host to generate a Swarm token

Before you can configure a Swarm, you start by provisioning a host with Engine.
Open a terminal on the host where you installed Machine. Then, to provision a
host called `local`, do the following:

```
docker-machine create -d virtualbox local
```

This examples uses VirtualBox but it could easily be DigitalOcean or a host on
your data center. The `local` value is the host name. Once you create it,
configure your terminal's shell environment to interact with the `local` host.

```
eval "$(docker-machine env local)"
```

Each Swarm host has a token installed into its Engine configuration. The token
allows the Swarm discovery backend to recognize a node as belonging to a
particular Swarm cluster. Create the token for your cluster by running the
`swarm` image:

```
docker run swarm create
Unable to find image 'swarm' locally
1.1.0-rc2: Pulling from library/swarm
892cb307750a: Pull complete
fe3c9860e6d5: Pull complete
cc01ef3f1fbc: Pull complete
b7e14a9c9c72: Pull complete
3ec746117013: Pull complete
703cb7acfce6: Pull complete
d4f6bb678158: Pull complete
2ad500e1bf96: Pull complete
Digest: sha256:f02993cd1afd86b399f35dc7ca0240969e971c92b0232a8839cf17a37d6e7009
Status: Downloaded newer image for swarm
0de84fa62a1d9e9cc2156111f63ac31f
```

The output of the `swarm create` command is a cluster token. Copy the token to a
safe place you will remember. Once you have the token, you can provision the
Swarm nodes and join them to the cluster_id.  The rest of this documentation,
refers to this token as the `SWARM_CLUSTER_TOKEN`.

## Provision Swarm nodes

All Swarm nodes in a cluster must have Engine installed. With Machine and the
`SWARM_CLUSTER_TOKEN` you can provision a host with Engine and configure it as a
Swarm node with one Machine command. To create a Swarm manager node on a new VM
called `swarm-manager`, you do the following:

```
docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-master \
    --swarm-discovery token://SWARM_CLUSTER_TOKEN \
    swarm-manager
```

Then, provision an additional node. You must supply the
`SWARM_CLUSTER_TOKEN` and a unique name for each host node, `HOST_NODE_NAME`.

```
docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery token://SWARM_CLUSTER_TOKEN \
    HOST_NODE_NAME
```

For example, you might use `node-01` as the `HOST_NODE_NAME` in the previous
example.

>**Note**: These command rely on Docker Swarm's hosted discovery service, Docker
Hub. If Docker Hub or your network is having issues, these commands may fail.
Check the [Docker Hub status page](http://status.docker.com/) for service
availability. If the problem Docker Hub, you can wait for it to recover or
configure other types of discovery backends.

## Connect node environments with Machine

If you are connecting to typical host environment with Machine, you use the
`env` subcommand, like this:

```
eval "$(docker-machine env local)"
```

Docker Machine provides a special `--swarm` flag with its `env` command to
connect to Swarm nodes.

```
docker-machine env --swarm HOST_NODE_NAME
export DOCKER_TLS_VERIFY="1"
export DOCKER_HOST="tcp://192.168.99.101:3376"
export DOCKER_CERT_PATH="/Users/mary/.docker/machine/machines/swarm-manager"
export DOCKER_MACHINE_NAME="swarm-manager"
# Run this command to configure your shell:
# eval $(docker-machine env --swarm HOST_NODE_NAME)
```

To set your SHELL connect to a Swarm node called `swarm-manager`, you would do
this:

```
eval "$(docker-machine env --swarm swarm-manager)"
```

Now, you can use the Docker CLI to query and interact with your cluster.

```
docker info
Containers: 2
Images: 1
Role: primary
Strategy: spread
Filters: health, port, dependency, affinity, constraint
Nodes: 1
 swarm-manager: 192.168.99.101:2376
  └ Status: Healthy
  └ Containers: 2
  └ Reserved CPUs: 0 / 1
  └ Reserved Memory: 0 B / 1.021 GiB
  └ Labels: executiondriver=native-0.2, kernelversion=4.1.13-boot2docker, operatingsystem=Boot2Docker 1.9.1 (TCL 6.4.1); master : cef800b - Fri Nov 20 19:33:59 UTC 2015, provider=virtualbox, storagedriver=aufs
CPUs: 1
Total Memory: 1.021 GiB
Name: swarm-manager
```

## Related information

* [Evaluate Swarm in a sandbox](install-w-machine.md)
* [Build a Swarm cluster for production](install-manual.md)
* [Swarm Discovery](discovery.md)
* [Docker Machine](https://docs.docker.com/machine) documentation
