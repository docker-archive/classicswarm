# Using Docker Swarm and Mesos

Swarm comes with a built-in scheduler that works with the Swarm manager to
schedule container resources. You can completely replace the built-in scheduler
with a 3rd party scheduler. For example, you can replace it with the Mesos
scheduler as described here.

When using Docker Swarm and Mesos, you use the Docker client to ask the Swarm
manager to schedule containers. The Swarm manager then schedules those
containers on a Mesos cluster.

###### The Docker Swarm on Mesos integration is experimental and **not**  production ready, please use with caution.

## Prerequisites

Each node in your Swarm must run a Mesos agent. The agent must be capable of
starting tasks in a Docker Container using the `--containerizers=docker` option.

You need to configure two TCP ports on the agent. One port to listen for the
Swarm manager, for example 2375. And a second TCP port to listen for the Mesos
master, for example 3375.

## Start the Docker Swarm manager

If you use a single Mesos master:

```     
$ docker run -d -p <swarm_port>:2375 -p 3375:3375 \
    swarm manage \
        -c mesos-experimental \
        --cluster-opt mesos.address=<public_machine_ip> \
        --cluster-opt mesos.port=3375 \
        <mesos_master_ip>:<mesos_master_port>
```

The command above creates a Swarm manager listening at `<swarm_port>`.
The `<mesos_master_ip>` value points to where Mesos master lives in the cluster.
Typically, this is localhost, a hostname, or an IP address.  The `<public_machine_ip>`
value is the IP address for Mesos master to talk to Swarm manager. If Mesos master
and Swarm manager are co-located on the same machine, you can use the `0.0.0.0`
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
Swarm container with an additional environment variable:
`SWARM_MESOS_USER=root`.

Exporting the environment variable `DOCKER_HOST=<swarm_ip>:<swarm_port>` will facilitate
interaction with the Docker client as it will not require `-H` anymore.


# Limitations

Docker Swarm on Mesos provides a basic Mesos offers negotiation
It is also possible to use Docker-compose to communicate to the Docker Swarm manager how to schedule Docker containers on the Mesos cluster.
Docker Swarm on Mesos supports Mesos master detection with Zookeeper

## Functionality

Docker Swarm on Mesos supports the following subset of Docker CLI commands:

+ attach
+ build
+ commit
+ cp
+ diff  
+ events
+ export
+ history
+ images
+ info
+ inspect
+ logs
+ network
  * create
  * remove
  * ls
  * inspect
  * connect
  * disconnect
+ port
+ ps
+ run
+ save
+ search
+ stats
+ top

## Unsupported docker commands

The current Swarm scheduler for Mesos implementation is not feature complete. Hence the following Docker CLI commands are unsupported and may cause unpredictable results:

+ create
+ import
+ kill
+ load
+ login
+ logout
+ pause
+ pull
+ push
+ rename
+ restart
+ rm 
+ rmi
+ start
+ stop
+ tag
+ unpause
+ update
+ volume
+ wait

## Known issues

- Docker Swarm on Mesos only uses unreserved resources; and if Mesos offers reserved resources, the role info is also ignored which is rejected by Mesos master. See [here](https://github.com/docker/swarm/issues/1618) for a proposal on letting Docker Swarm on Mesos use  both un-reserved/reserved resources.
- See [here](https://github.com/docker/swarm/issues/1619) for a proposal on letting Docker Swarm on Mesos use revocable resources.
- Restarted Mesos agents have flaky recovery in conjunction with Docker Swarm. When Mesos agents restart, Mesos master don’t always send offers for those agents to Docker Swarm. This issue seems to be solved with the Swarm 1.1 release.

To have a global view of the tracked issues please refer to
[Mesos issues in Github](https://github.com/docker/swarm/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+mesos).
