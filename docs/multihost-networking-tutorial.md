<!--[metadata]>
+++
title = "Multi-host networking deployment tutorial"
description = "Multi-host networking deployment tutorial"
keywords = ["docker, swarm, clustering,  networking, examples"]
[menu.main]
parent="workw_swarm"
weight=3
+++
<![end-metadata]-->

# Multi-host networking deployment tutorial
This step-by-step tutorial aims to walk you through a full Docker Swarm deployment with all the containers sharing a same LAN. Tutorial [first published here](https://www.auzias.net/en/docker-network-multihost/).

## Prerequisites
### Environment
This tutorial has been tested with:
  - docker engine 1.10.2.
  - Swarm server 1.1.3.
  - [Consul 0.6.3](http://consul.io/), needed for discovery for the [Swarm containers](https://docs.docker.com/swarm/discovery/).

### Discovery
Both Swarm agents and containers require a discovery mean; the former is needed to create the cluster itself, while the latter enables the multi-host networking. Note that the container discovery is not necessary if the Swarm containers do not need a common LAN.

#### Containers discovery
[Consul](http://consul.io/) is used for the multi-host networking discovery, any other key-value store can be used -- note that the engine supports [Consul](http://consul.io/), [etcd]https://coreos.com/etcd/, and [ZooKeeper](https://zookeeper.apache.org/). If you don't have yet any affinities you might want to read this [comparison post](http://technologyconversations.com/2015/09/08/service-discovery-zookeeper-vs-etcd-vs-consul/).

#### Swarm agents discovery
Static file or token can be used for the Swarm agent discovery. Tokens use REST API, and thus the network and an external server. The static file is preferred as it is faster, safer and also easier. Both methods are detailed below.

### Docker daemon
All nodes should run the Docker daemon as follow:

    /usr/bin/docker daemon -H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock --cluster-advertise eth0:2375 --cluster-store consul://127.0.0.1:8500

#### Options details:

    -H tcp://0.0.0.0:2375

Binds the daemon to an interface to allow be part of the [swarm](https://docs.docker.com/swarm/) cluster. An IP address can obviously be specified, it is a better solution if you have several NIC. The Docker daemons bind on the port 2375, the Swarm daemon will bind on the port 5732, see below.

    --cluster-advertise eth0:2375

Defines the interface and the port of the docker daemon should use to advertise itself.

    --cluster-store consul://127.0.0.1:8500

Defines the URI of the distributed kv storage backend. As consul is distributed the node can

As [consul](http://consul.io/) is distributed a local endpoint is used instead of the consul master. Hence the master can be changed without having to restart Docker daemon on all Swarm agents and the daemons communicate locally with the cluster-storage while the cluster-storages communicate together through the network. It also allows to select the Swarm manager after the daemon has been started.

### The sh-aliases used
In the following commands these two aliases are used:

    alias ldocker='docker -H tcp://0.0.0.0:2375'
    alias swarm-docker='docker -H tcp://0.0.0.0:5732' #used only on the swarm manager

Be sure to have the path of the consul binary in your `$PATH`. Once you are in the directory, typing `export PATH=$PATH:$(pwd)` will do the trick.
Swarm manager is assumed to be the consul master. It is also assumed that the variable `$IP` (local IP address) has been properly set and exported as well as `$MASTER_IP` (Swarm manager and consul master IP address). It must be notice that on the master, the variables `$IP` and `$MASTER_IP` are the same.

### The following
In the remaining of the tutorial these steps will be detailed:
 - [Consul](http://consul.io/) deployment (consul master, consul agent, and the joining).
 - Swarm configuration and starting (with both token and static file Swarm agents discovery).
 - Swarm network overlay configuration and an example on how to start debian containers sharing this network.

---

## Consul
Let's start to deploy all [consul](http://consul.io/) master and members as needed.

### Consul master
Start the consul master with the command line:

    user@master:~$ consul agent -server -bootstrap-expect 1 -data-dir /tmp/consul -node=master$HOSTNAME -bind=$MASTER_IP -client=$MASTER_IP

##### Options details:

    agent -server

Start the [consul](http://consul.io/) agent as a server.

    -bootstrap-expect 1

We expect only one master.

    -node=master$HOSTNAME

In this example, [consul](http://consul.io/) master will be named "mastermaster".

    -bind=$MASTER_IP

Specifies the IP address on which it should be bound. Optional if you have only one NIC.

    -client=$MASTER_IP

Specifies the RPC IP address on which the server should be bound. By default it is localhost. Note that this force the user to add `-rpc-addr=$IP:8400` for local request such as `consul members -rpc-addr=$IP:8400` or `consul join -rpc-addr=$IP:8400 192.168.196.9` to join the [consul](http://consul.io/) member that has the IP address `192.168.196.9`.

### Consul members
Start the consul agent on all Swarm agents/consul members with the command line:

    consul agent -data-dir /tmp/consul -node=$HOSTNAME -bind=$IP

As the options are close to the consul master command, they will not be detailed.

### Join consul members
In this tutorial, the consul members have an IP address in the network 192.168.196.0/24. They are all named bugsN with N being the last octet of the IP address.
    user@master:~$ consul join -rpc-addr=$MASTER_IP:8400 192.168.196.16
    user@master:~$ consul join -rpc-addr=$MASTER_IP:8400 192.168.196.17
    user@master:~$ consul join -rpc-addr=$MASTER_IP:8400 192.168.196.18
    user@master:~$ consul join -rpc-addr=$MASTER_IP:8400 192.168.196.19

It can be checked if the consul members are part of the [consul](http://consul.io/) deployment with the command:

    consul members -rpc-addr=$MASTER_IP:8400
    Node          Address              Status  Type    Build  Protocol  DC
    mastermaster  192.168.196.20:8301  alive   server  0.6.3  2         dc1
    bugs19        192.168.196.19:8301  alive   client  0.6.3  2         dc1
    bugs18        192.168.196.18:8301  alive   client  0.6.3  2         dc1
    bugs17        192.168.196.17:8301  alive   client  0.6.3  2         dc1
    bugs16        192.168.196.16:8301  alive   client  0.6.3  2         dc1

[Consul](http://consul.io/) members and master are deployed and working.

---
## Swarm
In the following the creation of swarm manager and swarm members discovery are detailed using two different methods: token and static file. Tokens use a hosted discovery service with Docker Hub while static file is just local and does not use the network. Static file solution should be preferred (and is actually easier).
### [static file] Start the swarm manager *while joining swarm members*
Create a file named `/tmp/cluster.disco` with the content `swarm_agent_ip:2375`.

    user@master:~$ cat /tmp/cluster.disco
    192.168.196.16:2375
    192.168.196.17:2375
    192.168.196.18:2375
    192.168.196.19:2375

Then just start the swarm manager as follow:

    user@master:~$ ldocker run -v /tmp/cluster.disco:/tmp/cluster.disco -d -p 5732:2375 --name swarm-master swarm manage file:///tmp/cluster.disco


And you're done !

### [token] Discovery with a token
The swarm manager will need to be created then started, then, just like consul, the Swam agents will need to be joined.

#### [token] Create and start the swarm manager
On the swarm master, create a swarm:

    user@master:~$ ldocker run --rm swarm create > swarm_id

The command creates a swarm and saves the token ID in the file `swarm_id` of the current directory. Once created, the swarm manager needs to be run as a daemon:

    user@master:~$ ldocker run -d -p 5732:2375 --name swarm-master swarm manage token://`cat swarm_id`

To check if it is well started, run the command:

    user@master:~$ ldocker ps
    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                    NAMES
    d28238445532        swarm               "/swarm manage token:"   5 seconds ago       Up 4 seconds        0.0.0.0:5732->2375/tcp   swarm-master

#### [token] Join swarm members into the swarm cluster
Then the swarm manager will need to join the Swarm agents.

    user@master:~$ ldocker run swarm join --addr=192.168.196.16:2375 token://`cat swarm_id`
    user@master:~$ ldocker run swarm join --addr=192.168.196.17:2375 token://`cat swarm_id`
    user@master:~$ ldocker run swarm join --addr=192.168.196.18:2375 token://`cat swarm_id`
    user@master:~$ ldocker run swarm join --addr=192.168.196.19:2375 token://`cat swarm_id`

std[in|out] will be busy, these commands need to be ran on different terminals. Adding `-d` before the `join` should solve this and enables a for-loop to be used for the joins.

To check if the joins has succeed, run the command:

    user@master:~$ ldocker ps
    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                    NAMES
    d1de6e4ee3fc        swarm               "/swarm join --addr=1"   5 seconds ago       Up 4 seconds        2375/tcp                 fervent_lichterman
    338572b87ce9        swarm               "/swarm join --addr=1"   6 seconds ago       Up 4 seconds        2375/tcp                 mad_ramanujan
    7083e4d6c7ea        swarm               "/swarm join --addr=1"   7 seconds ago       Up 5 seconds        2375/tcp                 naughty_sammet
    0c5abc6075da        swarm               "/swarm join --addr=1"   8 seconds ago       Up 6 seconds        2375/tcp                 gloomy_cray
    ab746399f106        swarm               "/swarm manage token:"   25 seconds ago      Up 23 seconds       0.0.0.0:5732->2375/tcp   swarm-master

### After the discovery of the swarm members
To check if the Docker cluster is healthy and running, run the command:

    user@master:~$ swarm-docker info
    Containers: 4
    Images: 4
    Role: primary
    Strategy: spread
    Filters: health, port, dependency, affinity, constraint
    Nodes: 4
     bugs16: 192.168.196.16:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 12
      └ Reserved Memory: 0 B / 49.62 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.16.0-4-amd64, operatingsystem=Debian GNU/Linux 8 (jessie), storagedriver=aufs
     bugs17: 192.168.196.17:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 12
      └ Reserved Memory: 0 B / 49.62 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.16.0-4-amd64, operatingsystem=Debian GNU/Linux 8 (jessie), storagedriver=aufs
     bugs18: 192.168.196.18:2375
      └ Containers: 0
      └ Reserved CPUs: 0 / 12
      └ Reserved Memory: 0 B / 49.62 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.16.0-4-amd64, operatingsystem=Debian GNU/Linux 8 (jessie), storagedriver=aufs
     bugs19: 192.168.196.19:2375
      └ Containers: 4
      └ Reserved CPUs: 0 / 12
      └ Reserved Memory: 0 B / 49.62 GiB
      └ Labels: executiondriver=native-0.2, kernelversion=3.16.0-4-amd64, operatingsystem=Debian GNU/Linux 8 (jessie), storagedriver=aufs
    CPUs: 48
    Total Memory: 198.5 GiB
    Name: ab746399f106


At this point swarm is deployed and all containers will be running over different nodes. Several containers can be deployed with the command:

    user@master:~$ swarm-docker run --rm -it debian bash

Once several time, run the command:

    user@master:~$ swarm-docker ps
    CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
    45b19d76d38e        debian              "bash"              6 seconds ago       Up 5 seconds                            bugs18/boring_mccarthy
    53e87693606e        debian              "bash"              6 seconds ago       Up 5 seconds                            bugs16/amazing_colden
    b18081f26a35        debian              "bash"              6 seconds ago       Up 4 seconds                            bugs17/small_newton
    f582d4af4444        debian              "bash"              7 seconds ago       Up 4 seconds                            bugs18/naughty_banach
    b3d689d749f9        debian              "bash"              7 seconds ago       Up 4 seconds                            bugs17/pensive_keller
    f9e86f609ffa        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs16/pensive_cray
    b53a46c01783        debian              "bash"              7 seconds ago       Up 4 seconds                            bugs18/reverent_ritchie
    78896a73191b        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs17/gloomy_bell
    a991d887a894        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs16/angry_swanson
    a43122662e92        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs17/pensive_kowalevski
    68d874bc19f9        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs16/modest_payne
    e79b3307f6e6        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs18/stoic_wescoff
    caac9466d86f        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs17/goofy_snyder
    7748d01d34ee        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs16/fervent_einstein
    99da2a91a925        debian              "bash"              7 seconds ago       Up 5 seconds                            bugs18/modest_goodall
    cd308099faac        debian              "bash"              7 seconds ago       Up 6 seconds                            bugs19/furious_ritchie

As shown, the containers are disseminated over bugs{16...19}.

---
## Multi-hosts network
A network overlay is needed so all the containers can be "plugged in" into the same virtual LAN. To create this network overlay, execute:

    user@master:~$ swarm-docker network create -d overlay net
    user@master:~$ swarm-docker network ls|grep "net"
    c96760503d06        net                 overlay

**And voilà !**

Once this overlay is created, add `--net=net` to the command `swarm-docker run --rm -it debian bash` and all your containers will be able to communicate natively as if they were on the same LAN. The default network is 10.0.0.0/24. The option `--subnet` allows to specify a subnetwork, i.e., `swarm-docker network create -d overlay --subnet 192.168.0.0/16 custom-subnet`. It then is possible to specify a static IP address to a container with the option `--ip`, i.e., `swarm-docker run --rm -it --net=custom-subnet --ip=192.168.0.13 debian bash`.
