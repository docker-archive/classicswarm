# Using Docker Swarm and Mesos

Swarm comes with a built-in scheduler that works with the swarm manager to
schedule container resources. You can completely replace the built-in scheduler
with a 3rd party scheduler. For example, you can replace it with the Mesos
scheduler as described here.

When using Docker Swarm and Mesos, you use the Docker client to ask the swarm
manager to schedule containers. The swarm manager then schedules those
containers on a Mesos cluster.

## Prerequisites

Each node in your swarm must run a Mesos slave. The slave must be capable of
starting tasks in a Docker Container using the `--containerizers=docker` option.

You need to configure two TCP ports on the slave. One port to listen for the
swarm manager, for example 2375. And a second TCP port to listen for the Mesos
master, for example 3375.

## Start the Docker Swarm manager

If you use a single Mesos master:

```     
$ docker run -d -p <swarm_port>:2375 -p 3375:3375 \
    swarm manage \
        -c mesos-experimental \
        --cluster-opt mesos.address=<public_machine_ip> \
        --cluster-opt mesos.port=3375 \
        <mesos_master_ip>:<mesos_master:port>
```

The command above creates a swarm manager listening at `<swarm_port>`.
The `<mesos_master_ip>` value points to where Mesos master lives in the cluster.
Typically, this is localhost, a hostname, or an IP address.  The `<public_machine_ip>`
value is the IP address for Mesos master to talk to swarm manager. If mesos master
and swarm manager are co-located on the same machine, you can use the `0.0.0.0`
or `localhost` value.

If you use multiple Mesos masters:

```
$ docker run -d -p <swarm_port>:2375 -p 3375:3375 \
    swarm manage \
        -c mesos-experimental \
        --cluster-opt mesos.address=<public_machine_ip> \
        --cluster-opt mesos.port=3375 \
        zk://<mesos_masters_url>
```

Once the manager is running, check your configuration by running `docker info`
as follows:

```
$ docker -H tcp://<swarm_ip>:<swarm_port> info
Containers: 0
Offers: 2
  Offer: 20150609-222929-1327399946-5050-14390-O6286
     └ cpus: 2
     └ mem: 1006 MiB
     └ disk: 34.37 GiB
     └ ports: 31000-32000
  Offer: 20150609-222929-1327399946-5050-14390-O6287
     └ cpus: 2
     └ mem: 1006 MiB
     └ disk: 34.37 GiB
     └ ports: 31000-32000
```

If you run into `Abnormal executor termination` error, you might want to run the
swarm container with an additional environment variable:
`SWARM_MESOS_USER=root`.
