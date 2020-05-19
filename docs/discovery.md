---
advisory: swarm-standalone
hide_from_sitemap: true
description: Swarm discovery
keywords: docker, swarm, clustering, discovery
title: Docker Swarm discovery
---

Docker Swarm comes with multiple discovery backends. You use a hosted discovery service with Docker Swarm. The service maintains a list of IPs in your cluster.
This page describes the different types of hosted discovery. These are:


## Use a distributed key/value store

Swarm node discovery supports the following backends:

* Consul 0.5.1 or higher
* Etcd 2.0 or higher
* ZooKeeper 3.4.5 or higher

### Use a hosted discovery key store

1. On each node, start the Swarm agent.

    The node IP address doesn't need to be public as long as the Swarm manager can access it. In a large cluster, the nodes joining swarm may trigger request spikes to discovery. For example, a large number of nodes are added by a script, or recovered from a network partition. This may result in discovery failure. You can use `--delay` option to specify a delay limit. The `swarm join` command adds a random delay less than this limit to reduce pressure to discovery.

    **Etcd**:

        swarm join --advertise=<node_ip:2375> etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>

    **Consul**:

        swarm join --advertise=<node_ip:2375> consul://<consul_addr>/<optional path prefix>

    **ZooKeeper**:

        swarm join --advertise=<node_ip:2375> zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>

2. Start the swarm manager on any machine or your laptop.

    **Etcd**:

        swarm manage -H tcp://<swarm_ip:swarm_port> etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>

    **Consul**:

        swarm manage -H tcp://<swarm_ip:swarm_port> consul://<consul_addr>/<optional path prefix>

    **ZooKeeper**:

        swarm manage -H tcp://<swarm_ip:swarm_port> zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>

4. Use the regular Docker commands.

        docker -H tcp://<swarm_ip:swarm_port> info
        docker -H tcp://<swarm_ip:swarm_port> run ...
        docker -H tcp://<swarm_ip:swarm_port> ps
        docker -H tcp://<swarm_ip:swarm_port> logs ...
        ...

5. Try listing the nodes in your cluster.

    **Etcd**:

        swarm list etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>
        <node_ip:2375>

    **Consul**:

        swarm list consul://<consul_addr>/<optional path prefix>
        <node_ip:2375>

    **ZooKeeper**:

        swarm list zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>
        <node_ip:2375>

### Use TLS with distributed key/value discovery

You can securely talk to the distributed k/v store using TLS. To connect
securely to the store, you must generate the certificates for a node when you
`join` it to the swarm. You can only use with Consul and Etcd. The following example illustrates this with Consul:

```
swarm join \
    --advertise=<node_ip:2375> \
    --discovery-opt kv.cacertfile=/path/to/mycacert.pem \
    --discovery-opt kv.certfile=/path/to/mycert.pem \
    --discovery-opt kv.keyfile=/path/to/mykey.pem \
    consul://<consul_addr>/<optional path prefix>
```

This works the same way for the swarm `manage` and `list` commands.

## A static file or list of nodes

> **Note**: This discovery method is incompatible with replicating swarm
managers. If you require replication, you should use a hosted discovery key
store.

You can use a static file or list of nodes for your discovery backend. The file must be stored on a host that is accessible from the swarm manager. You can also pass a node list as an option when you start Swarm.

Both the static file and the `nodes` option support an IP address range. To specify a range supply a pattern, for example, `10.0.0.[10:200]` refers to nodes starting from `10.0.0.10` to `10.0.0.200`. For example for the `file` discovery method.

        $ echo "10.0.0.[11:100]:2375"   >> /tmp/my_cluster
        $ echo "10.0.1.[15:20]:2375"    >> /tmp/my_cluster
        $ echo "192.168.1.2:[2:20]375"  >> /tmp/my_cluster

Or with node discovery:

        swarm manage -H <swarm_ip:swarm_port> "nodes://10.0.0.[10:200]:2375,10.0.1.[2:250]:2375"

### To create a file

1. Edit the file and add line for each of your nodes.

        echo <node_ip1:2375> >> /opt/my_cluster
        echo <node_ip2:2375> >> /opt/my_cluster
        echo <node_ip3:2375> >> /opt/my_cluster

    This example creates a file named `/tmp/my_cluster`. You can use any name you like.

2. Start the swarm manager on any machine.

        swarm manage -H tcp://<swarm_ip:swarm_port> file:///tmp/my_cluster

3. Use the regular Docker commands.

        docker -H tcp://<swarm_ip:swarm_port> info
        docker -H tcp://<swarm_ip:swarm_port> run ...
        docker -H tcp://<swarm_ip:swarm_port> ps
        docker -H tcp://<swarm_ip:swarm_port> logs ...
        ...

4. List the nodes in your cluster.

        $ swarm list file:///tmp/my_cluster
        <node_ip1:2375>
        <node_ip2:2375>
        <node_ip3:2375>

### To use a node list

1. Start the manager on any machine or your laptop.

        swarm manage -H <swarm_ip:swarm_port> nodes://<node_ip1:2375>,<node_ip2:2375>

    or

        swarm manage -H <swarm_ip:swarm_port> <node_ip1:2375>,<node_ip2:2375>

2. Use the regular Docker commands.

        docker -H <swarm_ip:swarm_port> info
        docker -H <swarm_ip:swarm_port> run ...
        docker -H <swarm_ip:swarm_port> ps
        docker -H <swarm_ip:swarm_port> logs ...

3. List the nodes in your cluster.

        $ swarm list file:///tmp/my_cluster
        <node_ip1:2375>
        <node_ip2:2375>
        <node_ip3:2375>


## Docker Hub as a hosted discovery service

> ### Deprecation Notice
>
> The Docker Hub Hosted Discovery Service has been removed on June 19th, 2019.
> Please switch to one of the other discovery mechanisms.
{:.warning}

## Docker Swarm documentation index

- [Docker Swarm overview](index.md)
- [Scheduler strategies](scheduler/strategy.md)
- [Scheduler filters](scheduler/filter.md)
- [Swarm API](swarm-api.md)
