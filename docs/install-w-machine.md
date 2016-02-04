<!--[metadata]>
+++
title = "Evaluate Swarm in a sandbox"
description = "Evaluate Swarm in a sandbox"
keywords = ["docker, swarm, clustering, discovery, examples"]
[menu.main]
parent="workw_swarm"
weight=-50
+++
<![end-metadata]-->

# Evaluate Swarm in a sandbox

This getting started example shows you how to create a Docker Swarm, the
native clustering tool for Docker.

You'll use Docker Toolbox to install Docker Machine and some other tools on your computer. Then you'll use Docker Machine to provision a set of Docker Engine hosts inside VirtualBox. Lastly, you'll use Docker Client to connect to the hosts, where you'll create a discovery token, compose a cluster of one Swarm manager and two nodes, and manage the cluster.

When you finish, you'll have a Docker Swarm up and running in VirtualBox on your
local Mac or Windows computer. You can use this Swarm as a personal development
sandbox.

To use Docker Swarm on Linux, see [Build a Swarm cluster for production](install-manual.md).

## Install Docker Toolbox

Download and install [Docker Toolbox](https://www.docker.com/docker-toolbox).

The toolbox installs a handful of tools on your local Windows or Mac OS X computer. In this exercise, you use three of those tools:

 - Docker Machine: To deploy virtual machines that run Docker Engine.
 - VirtualBox: To host the virtual machines deployed from Docker Machine.
 - Docker Client: To connect from your local computer to the Docker Engines on the VMs and issue docker commands to create the Swarm.

The following sections provide more information on each of these tools. The rest of the document uses the abbreviation, VM, for virtual machine.

## Create three VMs running Docker Engine

Here, you use Docker Machine to provision three VMs running Docker Engine.

1. Open a terminal on your computer. Use Docker Machine to list any VMs in VirtualBox.

		$ docker-machine ls
		NAME         ACTIVE   DRIVER       STATE     URL                         SWARM
		default    *        virtualbox   Running   tcp://192.168.99.100:2376

2. Optional: To conserve system resources, stop any virtual machines you are not using. For example, to stop the VM named `default`, enter:

		$ docker-machine stop default

3. Create and run a VM named `manager`.

		$ docker-machine create -d virtualbox manager

4. Create and run a VM named `agent1`.

		$ docker-machine create -d virtualbox node1_host

5. Create and run a VM named `agent2`.

		$ docker-machine create -d virtualbox node2_host


Each create command checks for a local copy of the *latest* VM image, called boot2docker.iso. If it isn't available, Docker Machine downloads the image from Docker Hub. Then, Docker Machine uses boot2docker.iso to create a VM that automatically runs Docker Engine.

> Troubleshooting: If your computer or hosts cannot reach Docker Hub, the
`docker-machine` or `docker run` commands that pull images may fail. In that
case, check the [Docker Hub status page](http://status.docker.com/) for
service availability. Then, check whether your computer is connected to the Internet.  Finally, check whether VirtualBox's network settings allow your hosts to connect to the Internet.

## Create a Swarm discovery token

Here you use the discovery backend hosted on Docker Hub to create a unique discovery token for your cluster. This discovery backend is only for low-volume development and testing purposes, not for production. Later on, when you run the Swarm manager and nodes, they register with the discovery backend as members of the cluster that's associated with the unique token. The discovery backend maintains an up-to-date list of cluster members and shares that list with the Swarm manager. The Swarm manager uses this list to assign tasks to the nodes.

1. Connect the Docker Client on your computer to the Docker Engine running on `mgr-host`.

        $ eval $(docker-machine env mgr-host)

    The client will send the `docker` commands in the following steps to the Docker Engine on on `mgr-host`.

2.  Create a unique id for the Swarm cluster.

        $ docker run --rm swarm create
        .
        .
        .
        Status: Downloaded newer image for swarm:latest
        dd80a29fd375e62dee5f8cd891d520db

    The `docker run` command gets the latest `swarm` image and runs it as a container. The `create` argument makes the Swarm container connect to the discovery backend hosted by Docker Hub and get a unique Swarm ID, also known as a "discovery token." The token appears only in the output; it is not saved to a file on the host. The `--rm` option automatically cleans up the container, including its file system, when the container finishes getting the token.

    The discovery service keeps unused tokens for approximately one week.

3. Copy the discovery token from the last line of the previous output to a safe place.

## Create the Swarm manager and nodes

Here, you connect to each of the hosts and create a Swarm manager or node.

1. Get the IP addresses of the three VMs. For example:

        $ docker-machine ls
        NAME      ACTIVE   DRIVER       STATE     URL                         SWARM   DOCKER   ERRORS
        agent1    -        virtualbox   Running   tcp://192.168.99.102:2376           v1.9.1
        agent2    -        virtualbox   Running   tcp://192.168.99.103:2376           v1.9.1
        manager   *        virtualbox   Running   tcp://192.168.99.100:2376           v1.9.1

2. Your Docker Client is connected to the Docker Engine on `manager`. Use the following syntax to run a Swarm container as the primary Swarm manager on `manager`.

        $ docker run -d -p <your_selected_port>:3376 -t -v /var/lib/boot2docker:/certs:ro swarm manage -H 0.0.0.0:3376 --tlsverify --tlscacert=/certs/ca.pem --tlscert=/certs/server.pem --tlskey=/certs/server-key.pem token://<cluster_id>

    For example:

        $ docker run -d -p 3376:3376 -t -v /var/lib/boot2docker:/certs:ro swarm manage -H 0.0.0.0:3376 --tlsverify --tlscacert=/certs/ca.pem --tlscert=/certs/server.pem --tlskey=/certs/server-key.pem swarm manage token://0ac50ef75c9739f5bfeeaf00503d4e6e

    The `-p` option maps a port 3376 on the container to port 3376 on the host. The `-v` option mounts the directory containing TLS certificates (`/var/lib/boot2docker` for the `manager` VM) into the container running Swarm manager in read-only mode.

3. Connect Docker Client to `nd1-host`.

        $ eval $(docker-machine env nd1-host)

4. Use the following syntax to run a Swarm container as a node on `nd1-host`. Replace <nodeip> with the IP address of the VM.

        $ docker run -d swarm join --addr=<nodeip>:<nodeport> token://<cluster_id>

    For example:

        $ docker run -d swarm join --addr=192.168.99.102:2376 token://0ac50ef75c9739f5bfeeaf00503d4e6e

5. Connect Docker Client to `nd2-host`.

        $ eval $(docker-machine env nd2-host)

6.  Run a Swarm container as an node on `nd2-host`. For example:

        $ docker run -d swarm join --addr=192.168.99.103:2376 token://0ac50ef75c9739f5bfeeaf00503d4e6e

## Manage your Swarm

Here, you connect to the cluster and review information about the Swarm manager and nodes. You tell the Swarm to run a container and check which node did the work.

1. Connect the Docker Client to the Swarm by updating the `DOCKER_HOST` environment variable.

        $ DOCKER_HOST=<manager_ip>:<your_selected_port>

    For the current example, the `manager` has IP address `192.168.99.100` and we selected port 3376 for the Swarm manager.

        $ DOCKER_HOST=192.168.99.100:3376

    Because Docker Swarm uses the standard Docker API, you can connect to it using  Docker Client and other tools such as Docker Compose, Dokku, Jenkins, and Krane, among others.

    For example:

        $ docker -H 192.168.99.101:3375 info

    If you're already connected to the manager host (e.g., `eval $(docker-machine env manager)`), you can drop the <manager_host_ip>. For example, to get cluster information, enter:

    To work around this problem, temporarily reconfigure Docker Client to not require TLS (This reconfiguration expires next time you use `eval $(docker-machine env <hostname>)`):

        $ unset DOCKER_TLS_VERIFY
        $ unset DOCKER_CERT_PATH

    Now, try again:

        $ docker -H 192.168.99.101:3375 info
        Containers: 0
         Running: 0
         Paused: 0
         Stopped: 0
        Images: 0
        Role: primary
        Strategy: spread
        Filters: health, port, dependency, affinity, constraint
        Nodes: 2
         (unknown): 192.168.99.102:3375
          └ Status: Pending
          └ Containers: 0
          └ Reserved CPUs: 0 / 0
          └ Reserved Memory: 0 B / 0 B
          └ Labels:
          └ Error: Cannot connect to the docker engine endpoint
          └ UpdatedAt: 2016-02-05T14:11:23Z
         (unknown): 192.168.99.103:3375
          └ Status: Pending
          └ Containers: 0
          └ Reserved CPUs: 0 / 0
          └ Reserved Memory: 0 B / 0 B
          └ Labels:
          └ Error: Cannot connect to the docker engine endpoint
          └ UpdatedAt: 2016-02-05T14:12:23Z
        Plugins:
         Volume:
         Network:
        Kernel Version: 4.1.17-boot2docker
        Operating System: linux
        Architecture: amd64
        CPUs: 0
        Total Memory: 0 B
        Name: 3ff295992cb9

    As you can see, the output displays information about the ***Swarm cluster***, including the manager and two nodes and the one manager node in the Swarm.

3. Check the images currently running on your Swarm.

        $ docker -H 192.168.99.101:3375  ps

4. Run a container on the Swarm.

        $ docker -H 192.168.99.101:3375 run hello-world
        Hello from Docker.
        .
        .
        .

5. Use the `docker ps` command to find out which node the container ran on. For example:

        $ docker ps -a
        CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS                      PORTS               NAMES
        0b0628349187        hello-world         "/hello"                 20 minutes ago      Exited (0) 20 minutes ago                       TBD
        .
        .
        .

    In this case, the Swarm ran `hello-world` on the `swarm1`.

    By default, Docker Swarm uses the "spread" strategy to choose which node runs a container. When you run multiple containers, the spread strategy assigns each container to the node with the fewest containers.

## Where to go next

At this point, you've done the following:
 - Created a Swarm discovery token.
 - Created Swarm nodes using Docker Machine.
 - Managed a Swarm and run containers on it.
 - Learned Swarm-related concepts and terminology.

However, Docker Swarm has many other aspects and capabilities.
For more information, visit [the Swarm landing page](https://www.docker.com/docker-swarm) or read the [Swarm documentation](https://docs.docker.com/swarm/).
