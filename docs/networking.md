<!--[metadata]>
+++
title = "Swarm and container networks"
description = "Swarm and container networks"
keywords = ["docker, swarm, clustering,  networking"]
[menu.main]
parent="workw_swarm"
weight=3
+++
<![end-metadata]-->

# Swarm and container networks

Docker Swarm is fully compatible with Docker's networking features. This
includes the multi-host networking feature which allows creation of custom
container networks that span multiple Docker hosts.

Before using Swarm with a custom network, read through the conceptual
information in [Docker container
networking](https://docs.docker.com/engine/userguide/networking/dockernetworks/).
You should also have walked through the [Get started with multi-host
networking](https://docs.docker.com/engine/userguide/networking/get-started-overlay/)
example.

## Create a custom network in a Swarm cluster

Multi-host networks require a key-value store. The key-value store holds
information about the network state which includes discovery, networks,
endpoints, IP addresses, and more. Through the Docker's libkv project, Docker
supports Consul, Etcd, and ZooKeeper key-value store backends. For details about
the supported backends, refer to the [libkv
project](https://github.com/docker/libkv).

To create a custom network, you must choose a key-value store backend and
implement it on your network. Then, you configure the Docker Engine daemon to
use this store. Two required parameters,  `--cluster-store` and
`--cluster-advertise`, refer to your key-value store server.

Once you've configured and restarted the daemon on each Swarm node, you are
ready to create a network.

## List networks

This example assumes there are two nodes `node-0` and `node-1` in the cluster.
From a Swarm node, list the networks:

```bash
$ docker network ls
NETWORK ID          NAME                   DRIVER
3dd50db9706d        node-0/host            host
09138343e80e        node-0/bridge          bridge
8834dbd552e5        node-0/none            null
45782acfe427        node-1/host            host
8926accb25fd        node-1/bridge          bridge
6382abccd23d        node-1/none            null
```

As you can see, each network name is prefixed by the node name.

## Create a network

By default, Swarm is using the `overlay` network driver, a global-scope network
driver. A global-scope network driver creates a network across an entire Swarm cluster.
When you create an `overlay` network under Swarm, you can omit the `-d` option:

```bash
$ docker network create swarm_network
42131321acab3233ba342443Ba4312
$ docker network ls
NETWORK ID          NAME                   DRIVER
3dd50db9706d        node-0/host            host
09138343e80e        node-0/bridge          bridge
8834dbd552e5        node-0/none            null
42131321acab        node-0/swarm_network   overlay
45782acfe427        node-1/host            host
8926accb25fd        node-1/bridge          bridge
6382abccd23d        node-1/none            null
42131321acab        node-1/swarm_network   overlay
```

As you can see here, both the `node-0/swarm_network` and the
`node-1/swarm_network` have the same ID.  This is because when you create a
network on the cluster, it is accessible from all the nodes.

To create a local scope network (for example with the `bridge` network driver) you
should use `<node>/<name>` otherwise your network is created on a random node.

```bash
$ docker network create node-0/bridge2 -b bridge
921817fefea521673217123abab223
$ docker network create node-1/bridge2 -b bridge
5262bbfe5616fef6627771289aacc2
$ docker network ls
NETWORK ID          NAME                   DRIVER
3dd50db9706d        node-0/host            host
09138343e80e        node-0/bridge          bridge
8834dbd552e5        node-0/none            null
42131321acab        node-0/swarm_network   overlay
921817fefea5        node-0/bridge2         bridge
45782acfe427        node-1/host            host
8926accb25fd        node-1/bridge          bridge
6382abccd23d        node-1/none            null
42131321acab        node-1/swarm_network   overlay
5262bbfe5616        node-1/bridge2         bridge
```

## Remove a network

To remove a network you can use its ID or its name. If two different networks
have the same name, include the `<node>` value:

```bash
$ docker network rm swarm_network
42131321acab3233ba342443Ba4312
$ docker network rm node-0/bridge2
921817fefea521673217123abab223
$ docker network ls
NETWORK ID          NAME                   DRIVER
3dd50db9706d        node-0/host            host
09138343e80e        node-0/bridge          bridge
8834dbd552e5        node-0/none            null
45782acfe427        node-1/host            host
8926accb25fd        node-1/bridge          bridge
6382abccd23d        node-1/none            null
5262bbfe5616        node-1/bridge2         bridge
```

The `swarm_network` was removed from every node. The `bridge2` was removed only
from `node-0`.

## Docker Swarm documentation index

- [Docker Swarm overview](index.md)
- [Scheduler strategies](scheduler/strategy.md)
- [Scheduler filters](scheduler/filter.md)
- [Swarm API](swarm-api.md)
