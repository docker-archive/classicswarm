---
advisory: swarm-standalone
description: Use overlay for multi-host networking
keywords: Examples, Usage, network, docker, documentation, user guide, multihost, cluster
title: Multi-host networking with standalone swarms
redirect_from:
- /engine/userguide/networking/overlay-standalone-swarm/
- /network/overlay-standalone.swarm/
---

## Standalone swarm only!

This article only applies to users who need to use a standalone swarm with
Docker, as opposed to swarm mode. Standalone swarms (sometimes known as Swarm
Classic) rely on an external key-value store to store networking information.
Docker swarm mode stores networking information in the Raft logs on the swarm
managers. If you use swarm mode, see
[swarm mode networking](../engine/swarm/networking.md) instead of this article.

If you are using standalone swarms, this article may be useful
to you. This article uses an example to explain the basics of creating a
multi-host network using a standalone swarm and the `overlay` network driver.
Unlike `bridge` networks, overlay networks require some pre-existing conditions
before you can create one:

## Overlay networking with an external key-value store

To use Docker with an external key-value store, you need the following:

* Access to the key-value store. Docker supports Consul, Etcd, and ZooKeeper
  (Distributed store) key-value stores. This example uses Consul.
* A cluster of hosts with connectivity to the key-value store.
* Docker running on each host in the cluster.
* Hosts within the cluster must have unique hostnames because the key-value
  store uses the hostnames to identify cluster members.

Docker Machine and Docker Swarm are not mandatory to experience Docker
multi-host networking with a key-value store. However, this example uses them to
illustrate how they are integrated. You use Machine to create both the
key-value store server and the host cluster using a standalone swarm.

>**Note**: These examples are not relevant to Docker running in swarm mode and
> do not work in such a configuration.

### Prerequisites

Before you begin, make sure you have a system on your network with the latest
version of Docker and Docker Machine installed. The example also relies
on VirtualBox. If you installed on a Mac or Windows with Docker Toolbox, you
have all of these installed already.

If you have not already done so, make sure you upgrade Docker and Docker
Machine to the latest versions.

### Set up a key-value store

An overlay network requires a key-value store. The key-value store holds
information about the network state which includes discovery, networks,
endpoints, IP addresses, and more. Docker supports Consul, Etcd, and ZooKeeper
key-value stores. This example uses Consul.

1.  Log into a system prepared with Docker and Docker Machine installed.

2.  Provision a VirtualBox machine called `mh-keystore`.

    ```bash
    $ docker-machine create -d virtualbox mh-keystore
    ```

    When you provision a new machine, the process adds Docker to the
    host. This means rather than installing Consul manually, you can create an
    instance using the [consul image from Docker
    Hub](https://hub.docker.com/_/consul/). You do this in the next step.

3.  Set your local environment to the `mh-keystore` machine.

    ```bash
    $  eval "$(docker-machine env mh-keystore)"
    ```

4.  Start a `consul` container running on the `mh-keystore` Docker machine.

    ```bash
    $  docker run -d \
      --name consul \
      -p "8500:8500" \
      -h "consul" \
      consul agent -server -bootstrap -client "0.0.0.0"
    ```

    The client starts a `consul` image running in the
    `mh-keystore` Docker machine. The server is called `consul` and is
    listening on port `8500`.

5.  Run the `docker ps` command to see the `consul` container.

    ```bash
    $ docker ps

    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                                                                      NAMES
    a47492d6c4d1        consul              "docker-entrypoint..."   2 seconds ago       Up 1 second         8300-8302/tcp, 8301-8302/udp, 8600/tcp, 8600/udp, 0.0.0.0:8500->8500/tcp   consul
    ```

Keep your terminal open and move on to
[Create a swarm cluster](#create-a-swarm-cluster).

### Create a swarm cluster

In this step, you use `docker-machine` to provision the hosts for your network.
You don't actually create the network yet. You create several
Docker machines in VirtualBox. One of the machines acts as the swarm manager
and you create that first. As you create each host, you pass the Docker
daemon on that machine options that are needed by the `overlay` network driver.

> **Note**: This creates a standalone swarm cluster, rather than using Docker
> in swarm mode. These examples are not relevant to Docker running in swarm mode
> and do not work in such a configuration.

1.  Create a swarm manager.

    ```bash
    $ docker-machine create \
    -d virtualbox \
    --swarm --swarm-master \
    --swarm-discovery="consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-store=consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    mhs-demo0
    ```

    At creation time, you supply the Docker daemon with the `--cluster-store`
    option. This option tells the Engine the location of the key-value store for
    the `overlay` network. The bash expansion `$(docker-machine ip mh-keystore)`
    resolves to the IP address of the Consul server you created in "STEP 1". The
    `--cluster-advertise` option advertises the machine on the network.

2.  Create another host and add it to the swarm.

    ```bash
    $ docker-machine create -d virtualbox \
      --swarm \
      --swarm-discovery="consul://$(docker-machine ip mh-keystore):8500" \
      --engine-opt="cluster-store=consul://$(docker-machine ip mh-keystore):8500" \
      --engine-opt="cluster-advertise=eth1:2376" \
      mhs-demo1
      ```

3.  List your Docker machines to confirm they are all up and running.

    ```bash
    $ docker-machine ls

    NAME         ACTIVE   DRIVER       STATE     URL                         SWARM
    default      -        virtualbox   Running   tcp://192.168.99.100:2376
    mh-keystore  *        virtualbox   Running   tcp://192.168.99.103:2376
    mhs-demo0    -        virtualbox   Running   tcp://192.168.99.104:2376   mhs-demo0 (master)
    mhs-demo1    -        virtualbox   Running   tcp://192.168.99.105:2376   mhs-demo0
    ```

At this point you have a set of hosts running on your network. You are ready to
create a multi-host network for containers using these hosts.

Leave your terminal open and go on to
[Create the overlay network](#create-the-overlay-network).

### Create the overlay network

To create an overlay network:

1.  Set your docker environment to the swarm manager.

    ```bash
    $ eval $(docker-machine env --swarm mhs-demo0)
    ```

    Using the `--swarm` flag with `docker-machine` restricts the `docker`
    commands to swarm information alone.

2.  Use the `docker info` command to view the swarm.

    ```bash
    $ docker info

    Containers: 3
    Images: 2
    Role: primary
    Strategy: spread
    Filters: affinity, health, constraint, port, dependency
    Nodes: 2
    mhs-demo0: 192.168.99.104:2376
    └ Containers: 2
    └ Reserved CPUs: 0 / 1
    └ Reserved Memory: 0 B / 1.021 GiB
    └ Labels: executiondriver=native-0.2, kernelversion=4.1.10-boot2docker, operatingsystem=Boot2Docker 1.9.0 (TCL 6.4); master : 4187d2c - Wed Oct 14 14:00:28 UTC 2015, provider=virtualbox, storagedriver=aufs
    mhs-demo1: 192.168.99.105:2376
    └ Containers: 1
    └ Reserved CPUs: 0 / 1
    └ Reserved Memory: 0 B / 1.021 GiB
    └ Labels: executiondriver=native-0.2, kernelversion=4.1.10-boot2docker, operatingsystem=Boot2Docker 1.9.0 (TCL 6.4); master : 4187d2c - Wed Oct 14 14:00:28 UTC 2015, provider=virtualbox, storagedriver=aufs
    CPUs: 2
    Total Memory: 2.043 GiB
    Name: 30438ece0915
    ```

    This output shows that you are running three containers and two images on
    the manager.

3.  Create your `overlay` network.

    ```bash
    $ docker network create --driver overlay --subnet=10.0.9.0/24 my-net
    ```

    You only need to create the network on a single host in the cluster. In this
    case, you used the swarm manager but you could easily have run it on any
    host in the swarm.

    > **Note**: It is highly recommended to use the `--subnet` option when creating
    > a network. If the `--subnet` is not specified, Docker automatically
    > chooses and assigns a subnet for the network and it could overlap with another subnet
    > in your infrastructure that is not managed by Docker. Such overlaps can cause
    > connectivity issues or failures when containers are connected to that network.

4.  Check that the network exists:

    ```bash
    $ docker network ls

    NETWORK ID          NAME                DRIVER
    412c2496d0eb        mhs-demo1/host      host
    dd51763e6dd2        mhs-demo0/bridge    bridge
    6b07d0be843f        my-net              overlay
    b4234109bd9b        mhs-demo0/none      null
    1aeead6dd890        mhs-demo0/host      host
    d0bb78cbe7bd        mhs-demo1/bridge    bridge
    1c0eb8f69ebb        mhs-demo1/none      null
    ```

    Since you are in the swarm manager environment, you see all the networks on all
    the swarm participants: the default networks on each Docker daemon and the single overlay
    network. Each network has a unique ID and a namespaced name.

5.  Switch to each swarm agent in turn and list the networks.

    ```bash
    $ eval $(docker-machine env mhs-demo0)

    $ docker network ls

    NETWORK ID          NAME                DRIVER
    6b07d0be843f        my-net              overlay
    dd51763e6dd2        bridge              bridge
    b4234109bd9b        none                null
    1aeead6dd890        host                host


    $ eval $(docker-machine env mhs-demo1)

    $ docker network ls

    NETWORK ID          NAME                DRIVER
    d0bb78cbe7bd        bridge              bridge
    1c0eb8f69ebb        none                null
    412c2496d0eb        host                host
    6b07d0be843f        my-net              overlay
    ```

Both agents report they have the `my-net` network with the `6b07d0be843f` ID.
You now have a multi-host container network running!

### Run an application on your network

Once your network is created, you can start a container on any of the hosts and
it automatically is part of the network.

1.  Set your environment to the swarm manager.

    ```bash
    $ eval $(docker-machine env --swarm mhs-demo0)
    ```

2.  Start an Nginx web server on the `mhs-demo0` instance.

    ```bash
    $ docker run -itd \
      --name=web \
      --network=my-net \
      --env="constraint:node==mhs-demo0" \
      nginx:alpine
    ```

4.  Run a `busybox` instance on the `mhs-demo1` instance and get the contents of
    the Nginx server's home page.

    ```none
    $ docker run -it --rm \
      --network=my-net \
      --env="constraint:node==mhs-demo1" \
      busybox wget -O- http://web

    Unable to find image 'busybox:latest' locally
    latest: Pulling from library/busybox
    ab2b8a86ca6c: Pull complete
    2c5ac3f849df: Pull complete
    Digest: sha256:5551dbdfc48d66734d0f01cafee0952cb6e8eeecd1e2492240bf2fd9640c2279
    Status: Downloaded newer image for busybox:latest
    Connecting to web (10.0.9.2:80)
    <!DOCTYPE html>
    <html>
    <head>
    <title>Welcome to nginx!</title>
    <style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
    </style>
    </head>
    <body>
    <h1>Welcome to nginx!</h1>
    <p>If you see this page, the nginx web server is successfully installed and
    working. Further configuration is required.</p>

    <p>For online documentation and support, refer to
    <a href="http://nginx.org/">nginx.org</a>.<br/>
    Commercial support is available at
    <a href="http://nginx.com/">nginx.com</a>.</p>

    <p><em>Thank you for using nginx.</em></p>
    </body>
    </html>
    -                    100% |*******************************|   612   0:00:00 ETA
    ```

### Check external connectivity

As you've seen, Docker's built-in overlay network driver provides out-of-the-box
connectivity between the containers on multiple hosts within the same network.
Additionally, containers connected to the multi-host network are automatically
connected to the `docker_gwbridge` network. This network allows the containers
to have external connectivity outside of their cluster.

1.  Change your environment to the swarm agent.

    ```bash
    $ eval $(docker-machine env mhs-demo1)
    ```

2.  View the `docker_gwbridge` network, by listing the networks.

    ```bash
    $ docker network ls

    NETWORK ID          NAME                DRIVER
    6b07d0be843f        my-net              overlay
    dd51763e6dd2        bridge              bridge
    b4234109bd9b        none                null
    1aeead6dd890        host                host
    e1dbd5dff8be        docker_gwbridge     bridge
    ```

3.  Repeat steps 1 and 2 on the swarm manager.

    ```bash
    $ eval $(docker-machine env mhs-demo0)

    $ docker network ls

    NETWORK ID          NAME                DRIVER
    6b07d0be843f        my-net              overlay
    d0bb78cbe7bd        bridge              bridge
    1c0eb8f69ebb        none                null
    412c2496d0eb        host                host
    97102a22e8d2        docker_gwbridge     bridge
    ```

2.  Check the Nginx container's network interfaces.

    ```bash
    $ docker container exec web ip addr

    1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
        valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
        valid_lft forever preferred_lft forever
    22: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP group default
    link/ether 02:42:0a:00:09:03 brd ff:ff:ff:ff:ff:ff
    inet 10.0.9.2/24 scope global eth0
        valid_lft forever preferred_lft forever
    inet6 fe80::42:aff:fe00:903/64 scope link
        valid_lft forever preferred_lft forever
    24: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff
    inet 172.18.0.2/16 scope global eth1
        valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link
        valid_lft forever preferred_lft forever
    ```

   The `eth0` interface represents the container interface that is connected to
   the `my-net` overlay network. While the `eth1` interface represents the
   container interface that is connected to the `docker_gwbridge` network.

## Use Docker Compose with swarm classic

Refer to the Networking feature introduced in
[Compose V2 format](../compose/networking.md)
and execute the multi-host networking scenario in the swarm cluster used above.

## Next steps

- [Networking overview](../network/index.md)
- [Overlay networks](../network/overlay.md)
