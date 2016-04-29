<!--[metadata]>
+++
title = "Discovery"
description = "Swarm discovery"
keywords = ["docker, swarm, clustering,  discovery"]
[menu.main]
parent="workw_swarm"
weight=4
+++
<![end-metadata]-->

# Docker Swarm Discovery

Docker Swarm comes with multiple discovery backends. You use a hosted discovery service with Docker Swarm. The service maintains a list of IPs in your cluster.
This page describes the different types of hosted discovery available to you. These are:


## Using a distributed key/value store

The recommended way to do node discovery in Swarm is Docker's libkv project. The libkv project is an abstraction layer over existing distributed key/value stores.  As of this writing, the project supports:

* Consul 0.5.1 or higher
* Etcd 2.0 or higher
* ZooKeeper 3.4.5 or higher

For details about libkv and a detailed technical overview of the supported backends, refer to the [libkv project](https://github.com/docker/libkv).

### Using a hosted discovery key store

1. On each node, start the Swarm agent.

    The node IP address doesn't have to be public as long as the Swarm manager can access it. In a large cluster, the nodes joining swarm may trigger request spikes to discovery. For example, a large number of nodes are added by a script, or recovered from a network partition. This may result in discovery failure. You can use `--delay` option to specify a delay limit. Swarm join will add a random delay less than this limit to reduce pressure to discovery.

    **Etcd**:

        swarm join --advertise=<node_ip:2375> etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>

    **Consul**:

        swarm join --advertise=<node_ip:2375> consul://<consul_addr>/<optional path prefix>

    **ZooKeeper**:

        swarm join --advertise=<node_ip:2375> zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>

2. Start the Swarm manager on any machine or your laptop.

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

This works the same way for the Swarm `manage` and `list` commands.

## A static file or list of nodes

You can use a static file or list of nodes for your discovery backend. The file must be stored on a host that is accessible from the Swarm manager. You can also pass a node list as an option when you start Swarm.

Both the static file and the `nodes` option support a IP address ranges. To specify a range supply a pattern, for example, `10.0.0.[10:200]` refers to nodes starting from `10.0.0.10` to `10.0.0.200`.  For example for the `file` discovery method.

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

2. Start the Swarm manager on any machine.

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

> **Warning**: The Docker Hub Hosted Discovery Service **is not recommended** for production use. It's intended to be used for testing/development. See the  discovery backends for production use.

This example uses the hosted discovery service on Docker Hub. Using
Docker Hub's hosted discovery service requires that each node in the
swarm is connected to the public internet. To create your cluster:

1. Create a cluster.

        $ swarm create
        6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>

2. Create each node and join them to the cluster.

    On each of your nodes, start the swarm agent. The <node_ip> doesn't have to be public (eg. 192.168.0.X) but the the Swarm manager must be able to access it.

        $ swarm join --advertise=<node_ip:2375> token://<cluster_id>

3. Start the Swarm manager.

    This can be on any machine or even your laptop.

        $ swarm manage -H tcp://<swarm_ip:swarm_port> token://<cluster_id>

4. Use regular Docker commands to interact with your cluster.

        docker -H tcp://<swarm_ip:swarm_port> info
        docker -H tcp://<swarm_ip:swarm_port> run ...
        docker -H tcp://<swarm_ip:swarm_port> ps
        docker -H tcp://<swarm_ip:swarm_port> logs ...
        ...

5. List the nodes in your cluster.

        swarm list token://<cluster_id>
        <node_ip:2375>

## Contribute a new discovery backend

You can contribute a new discovery backend to Swarm. For information on how to
do this, see <a
href="https://github.com/docker/docker/tree/master/pkg/discovery">
github.com/docker/docker/pkg/discovery</a>.


## Docker Swarm documentation index

- [Docker Swarm overview](index.md)
- [Scheduler strategies](scheduler/strategy.md)
- [Scheduler filters](scheduler/filter.md)
- [Swarm API](swarm-api.md)
