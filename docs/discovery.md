<!--[metadata]>
+++
title = "Docker Swarm discovery"
description = "Swarm discovery"
keywords = ["docker, swarm, clustering,  discovery"]
[menu.main]
parent="smn_workw_swarm"
weight=3
+++
<![end-metadata]-->

# Discovery

Docker Swarm comes with multiple Discovery backends.

## Backends

### Hosted Discovery with Docker Hub

First we create a cluster.

    # create a cluster
    $ swarm create
    6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>


Then we create each node and join them to the cluster.

    # on each of your nodes, start the swarm agent
    #  <node_ip> doesn't have to be public (eg. 192.168.0.X),
    #  as long as the swarm manager can access it.
    $ swarm join --advertise=<node_ip:2375> token://<cluster_id>


Finally, we start the Swarm manager. This can be on any machine or even
your laptop.

    $ swarm manage -H tcp://<swarm_ip:swarm_port> token://<cluster_id>

You can then use regular Docker commands to interact with your swarm.

    docker -H tcp://<swarm_ip:swarm_port> info
    docker -H tcp://<swarm_ip:swarm_port> run ...
    docker -H tcp://<swarm_ip:swarm_port> ps
    docker -H tcp://<swarm_ip:swarm_port> logs ...
    ...


You can also list the nodes in your cluster.

    swarm list token://<cluster_id>
    <node_ip:2375>


### Using a static file describing the cluster

For each of your nodes, add a line to a file. The node IP address
doesn't need to be public as long the Swarm manager can access it.

    echo <node_ip1:2375> >> /tmp/my_cluster
    echo <node_ip2:2375> >> /tmp/my_cluster
    echo <node_ip3:2375> >> /tmp/my_cluster


Then start the Swarm manager on any machine.

    swarm manage -H tcp://<swarm_ip:swarm_port> file:///tmp/my_cluster


And then use the regular Docker commands.

    docker -H tcp://<swarm_ip:swarm_port> info
    docker -H tcp://<swarm_ip:swarm_port> run ...
    docker -H tcp://<swarm_ip:swarm_port> ps
    docker -H tcp://<swarm_ip:swarm_port> logs ...
    ...

You can list the nodes in your cluster.

    $ swarm list file:///tmp/my_cluster
    <node_ip1:2375>
    <node_ip2:2375>
    <node_ip3:2375>


### Using etcd

On each of your nodes, start the Swarm agent. The node IP address
doesn't have to be public as long as the swarm manager can access it.

    swarm join --advertise=<node_ip:2375> etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>


Start the manager on any machine or your laptop.

    swarm manage -H tcp://<swarm_ip:swarm_port> etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>


And then use the regular Docker commands.

    docker -H tcp://<swarm_ip:swarm_port> info
    docker -H tcp://<swarm_ip:swarm_port> run ...
    docker -H tcp://<swarm_ip:swarm_port> ps
    docker -H tcp://<swarm_ip:swarm_port> logs ...
    ...


You can list the nodes in your cluster.

    swarm list etcd://<etcd_addr1>,<etcd_addr2>/<optional path prefix>
    <node_ip:2375>


### Using consul

On each of your nodes, start the Swarm agent. The node IP address
doesn't need to be public as long as the Swarm manager can access it.

    swarm join --advertise=<node_ip:2375> consul://<consul_addr>/<optional path prefix>

Start the manager on any machine or your laptop.

    swarm manage -H tcp://<swarm_ip:swarm_port> consul://<consul_addr>/<optional path prefix>


And then use the regular Docker commands.

    docker -H tcp://<swarm_ip:swarm_port> info
    docker -H tcp://<swarm_ip:swarm_port> run ...
    docker -H tcp://<swarm_ip:swarm_port> ps
    docker -H tcp://<swarm_ip:swarm_port> logs ...
    ...

You can list the nodes in your cluster.

    swarm list consul://<consul_addr>/<optional path prefix>
    <node_ip:2375>


### Using zookeeper

On each of your nodes, start the Swarm agent. The node IP doesn't have
to be public as long as the swarm manager can access it.

    swarm join --advertise=<node_ip:2375> zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>


Start the manager on any machine or your laptop.

    swarm manage -H tcp://<swarm_ip:swarm_port> zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>

You can then use the regular Docker commands.


    docker -H tcp://<swarm_ip:swarm_port> info
    docker -H tcp://<swarm_ip:swarm_port> run ...
    docker -H tcp://<swarm_ip:swarm_port> ps
    docker -H tcp://<swarm_ip:swarm_port> logs ...
    ...


You can list the nodes in the cluster.

    swarm list zk://<zookeeper_addr1>,<zookeeper_addr2>/<optional path prefix>
    <node_ip:2375>


### Using a static list of IP addresses

Start the manager on any machine or your laptop

    swarm manage -H <swarm_ip:swarm_port> nodes://<node_ip1:2375>,<node_ip2:2375>

Or

    swarm manage -H <swarm_ip:swarm_port> <node_ip1:2375>,<node_ip2:2375>


Then use the regular Docker commands.

    docker -H <swarm_ip:swarm_port> info
    docker -H <swarm_ip:swarm_port> run ...
    docker -H <swarm_ip:swarm_port> ps
    docker -H <swarm_ip:swarm_port> logs ...


### Range pattern for IP addresses

The `file` and `nodes` discoveries support a range pattern to specify IP
addresses, i.e., `10.0.0.[10:200]` will be a list of nodes starting from
`10.0.0.10` to `10.0.0.200`.

For example for the `file` discovery method.

    $ echo "10.0.0.[11:100]:2375"   >> /tmp/my_cluster
    $ echo "10.0.1.[15:20]:2375"    >> /tmp/my_cluster
    $ echo "192.168.1.2:[2:20]375"  >> /tmp/my_cluster

Then start the manager.

    swarm manage -H tcp://<swarm_ip:swarm_port> file:///tmp/my_cluster


And for the `nodes` discovery method.

    swarm manage -H <swarm_ip:swarm_port> "nodes://10.0.0.[10:200]:2375,10.0.1.[2:250]:2375"


## Contributing a new discovery backend

You can contribute a new discovery backend to Swarm. For information on how to do this, see [our discovery README in the Docker Swarm repository](https://github.com/docker/swarm/blob/master/discovery/README.md).

## Docker Swarm documentation index

- [User guide](index.md)
- [Sheduler strategies](/scheduler/strategy.md)
- [Sheduler filters](/scheduler/filter.md)
- [Swarm API](/api/swarm-api.md)
