<!--[metadata]>
+++
title = "Docker Swarm Networking"
description = "Swarm Networking"
keywords = ["docker, swarm, clustering,  networking"]
[menu.main]
parent="smn_workw_swarm"
weight=4
+++
<![end-metadata]-->

# Networking

Docker Swarm is fully compatible for the new networking model added in docker 1.9

## Setup

To use multi-host networking you need to start your docker engines with
`--cluster-store` and `--cluster-advertise` as indicated in the docker
engine docs.

### List networks

This example assumes there are two nodes `node-0` and `node-1` in the cluster.

    $ docker network ls
    NETWORK ID          NAME                   DRIVER
    3dd50db9706d        node-0/host            host
    09138343e80e        node-0/bridge          bridge
    8834dbd552e5        node-0/none            null
    45782acfe427        node-1/host            host
    8926accb25fd        node-1/bridge          bridge
    6382abccd23d        node-1/none            null

As you can see, each network name is prefixed by the node name.

## Create a network

By default, swarm is using the `overlay` network driver, a global
scope driver.

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

As you can see here, the ID is the same on the two nodes, because it's the same
network.

If you want to create a local scope network (for example with the bridge
driver) you should use `<node>/<name>`. Otherwise your network will be created on a
random node.

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
    921817fefea5        node-0/bridge2         brige
    45782acfe427        node-1/host            host
    8926accb25fd        node-1/bridge          bridge
    6382abccd23d        node-1/none            null
    42131321acab        node-1/swarm_network   overlay
    5262bbfe5616        node-1/bridge2         bridge

## Remove a network

To remove a network you can use its ID or its name.
If two different networks have the same name, you can use `<node>/<name>`.

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
    
`swarm_network` was removed from every node, whereas `bridge2` was removed only
from `node-0`.

## Docker Swarm documentation index

- [User guide](index.md)
- [Scheduler strategies](scheduler/strategy.md)
- [Scheduler filters](scheduler/filter.md)
- [Swarm API](api/swarm-api.md)
