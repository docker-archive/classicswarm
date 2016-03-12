<!--[metadata]>
+++
title = "High availability in Swarm"
description = "High availability in Swarm"
keywords = ["docker, swarm,  clustering"]
[menu.main]
parent="workw_swarm"
weight=3
+++
<![end-metadata]-->

# High availability in Docker Swarm

In Docker Swarm, the **Swarm manager** is responsible for the entire cluster and manages the resources of multiple *Docker hosts* at scale. If the Swarm manager dies, you must create a new one and deal with an interruption of service.

The *High Availability* feature allows a Docker Swarm to gracefully handle the failover of a manager instance. Using this feature, you can create a single **primary manager** instance and multiple **replica** instances.

A primary manager is the main point of contact with the Docker Swarm cluster. You can also create and talk to replica instances that will act as backups. Requests issued on a replica are automatically proxied to the primary manager. If the primary manager fails, a replica takes away the lead. In this way, you always keep a point of contact with the cluster.

## Setup primary and replicas

This section explains how to set up Docker Swarm using multiple **managers**.

### Assumptions

You need either a `Consul`, `etcd`, or `Zookeeper` cluster. This procedure is written assuming a `Consul` server running on address `192.168.42.10:8500`. All hosts will have a Docker Engine configured to listen on port 2375.  We will be configuring the Managers to operate on port 4000. The sample Swarm configuration has three machines:

- `manager-1` on `192.168.42.200`
- `manager-2` on `192.168.42.201`
- `manager-3` on `192.168.42.202`

### Create the primary manager

You use the `swarm manage` command with the `--replication` and `--advertise` flags to create a primary manager.

      user@manager-1 $ swarm manage -H :4000 <tls-config-flags> --replication --advertise 192.168.42.200:4000 consul://192.168.42.10:8500/nodes
      INFO[0000] Listening for HTTP addr=:4000 proto=tcp
      INFO[0000] Cluster leadership acquired
      INFO[0000] New leader elected: 192.168.42.200:4000
      [...]


The  `--replication` flag tells Swarm that the manager is part of a multi-manager configuration and that this primary manager competes with other manager instances for the primary role. The primary manager has the authority to manage cluster, replicate logs, and replicate events happening inside the cluster.

The `--advertise` option specifies the primary manager address. Swarm uses this address to advertise to the cluster when the node is elected as the primary. As you see in the command's output, the address you provided now appears to be the one of the elected Primary manager.


### Create two replicas

Now that you have a primary manager, you can create replicas.

    user@manager-2 $ swarm manage -H :4000 <tls-config-flags> --replication --advertise 192.168.42.201:4000 consul://192.168.42.10:8500/nodes
    INFO[0000] Listening for HTTP                            addr=:4000 proto=tcp
    INFO[0000] Cluster leadership lost
    INFO[0000] New leader elected: 192.168.42.200:4000
    [...]

This command creates a replica manager on `192.168.42.201:4000` which is looking at `192.168.42.200:4000` as the primary manager.

Create an additional, third *manager* instance:

    user@manager-3 $ swarm manage -H :4000 <tls-config-flags> --replication --advertise 192.168.42.202:4000 consul://192.168.42.10:8500/nodes
    INFO[0000] Listening for HTTP                            addr=:4000 proto=tcp
    INFO[0000] Cluster leadership lost
    INFO[0000] New leader elected: 192.168.42.200:4000
    [...]

Once you have established your primary manager and the replicas, create **Swarm agents** as you normally would.


### List machines in the cluster

Typing `docker info` should give you an output similar to the following:


    user@my-machine $ export DOCKER_HOST=192.168.42.200:4000 # Points to manager-1
    user@my-machine $ docker info
    Containers: 0
    Images: 25
    Storage Driver:
    Role: Primary  <--------- manager-1 is the Primary manager
    Primary: 192.168.42.200
    Strategy: spread
    Filters: affinity, health, constraint, port, dependency
    Nodes: 3
     swarm-agent-0: 192.168.42.100:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.053 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-49-generic, operatingsystem=Ubuntu 14.04.2 LTS, storagedriver=aufs
     swarm-agent-1: 192.168.42.101:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.053 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-49-generic, operatingsystem=Ubuntu 14.04.2 LTS, storagedriver=aufs
     swarm-agent-2: 192.168.42.102:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.053 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.13.0-49-generic, operatingsystem=Ubuntu 14.04.2 LTS, storagedriver=aufs
    Execution Driver:
    Kernel Version:
    Operating System:
    CPUs: 3
    Total Memory: 6.158 GiB
    Name:
    ID:
    Http Proxy:
    Https Proxy:
    No Proxy:

This information shows that `manager-1` is the current primary and supplies the address to use to contact this primary.

## Test the failover mechanism

To test the failover mechanism, you shut down the designated primary manager.
Issue a `Ctrl-C` or `kill` the current primary manager (`manager-1`) to shut it down.

### Wait for automated failover

After a short time, the other instances detect the failure and a replica takes the *lead* to become the primary manager.

For example, look at `manager-2`'s logs:

    user@manager-2 $ swarm manage -H :4000 <tls-config-flags> --replication --advertise 192.168.42.201:4000 consul://192.168.42.10:8500/nodes
    INFO[0000] Listening for HTTP                            addr=:4000 proto=tcp
    INFO[0000] Cluster leadership lost
    INFO[0000] New leader elected: 192.168.42.200:4000
    INFO[0038] New leader elected: 192.168.42.201:4000
    INFO[0038] Cluster leadership acquired               <--- We have been elected as the new Primary Manager
    [...]

Because the primary manager, `manager-1`, failed right after it was elected, the replica with the address `192.168.42.201:4000`, `manager-2`, recognized the failure and attempted to take away the lead. Because `manager-2` was fast enough, the process was effectively elected as the primary manager. As a result, `manager-2` became the primary manager of the cluster.

If we take a look at `manager-3` we should see those `logs`:

    user@manager-3 $ swarm manage -H :4000 <tls-config-flags> --replication --advertise 192.168.42.202:4000 consul://192.168.42.10:8500/nodes
    INFO[0000] Listening for HTTP                            addr=:4000 proto=tcp
    INFO[0000] Cluster leadership lost
    INFO[0000] New leader elected: 192.168.42.200:4000
    INFO[0036] New leader elected: 192.168.42.201:4000   <--- manager-2 sees the new Primary Manager
    [...]


At this point, we need to export the new `DOCKER_HOST` value.


### Switch the primary

To switch the `DOCKER_HOST` to use `manager-2` as the primary, you do the following:

    user@my-machine $ export DOCKER_HOST=192.168.42.201:4000 # Points to manager-2
    user@my-machine $ docker info
    Containers: 0
    Images: 25
    Storage Driver:
    Role: Replica  <--------- manager-2 is a Replica
    Primary: 192.168.42.200
    Strategy: spread
    Filters: affinity, health, constraint, port, dependency
    Nodes: 3

You can use the `docker` command on any Docker Swarm primary manager or any replica.

If you like, you can use custom mechanisms to always point `DOCKER_HOST` to the current primary manager. Then, you never lose contact with your Docker Swarm in the event of a failover.
