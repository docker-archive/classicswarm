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

You'll use Docker Toolbox to install Docker Machine and some other tools on your computer. Then you'll use Docker Machine to provision a set of Docker Engine hosts. Lastly, you'll use Docker client to connect to the hosts, where you'll create a discovery token, create a cluster of one Swarm manager and two nodes, and manage the cluster.

When you finish, you'll have a Docker Swarm up and running in VirtualBox on your
local Mac or Windows computer. You can use this Swarm as personal development
sandbox.

To use Docker Swarm on Linux, see [Build a Swarm cluster for production](install-manual.md).

## Install Docker Toolbox

Download and install [Docker Toolbox](https://www.docker.com/docker-toolbox).

The toolbox installs a handful of tools on your local Windows or Mac OS X computer. In this exercise, you use three of those tools:

 - Docker Machine: To deploy virtual machines that run Docker Engine.
 - VirtualBox: To host the virtual machines deployed from Docker Machine.
 - Docker Client: To connect from your local computer to the Docker Engines on the VMs and issue docker commands to manage the Swarm.

The following sections provide more information on each of these tools. The rest of the document uses the abbreviation, VM, for virtual machine.

## Verify that Docker Machine works

1. Open a terminal on your computer. Use Docker Machine to list any existing VMs in VirtualBox.

		$ docker-machine ls
		NAME         ACTIVE   DRIVER       STATE     URL                         SWARM
		default    *        virtualbox   Running   tcp://192.168.99.100:2376

2. Optional: To conserve system resources, stop any virtual machines you are not using. For example, to stop the VM named `default`, enter:

		$ docker-machine stop default

## Create a Swarm discovery token

To identify different clusters, Swarm uses something called a "discovery token". To create a unique one for your own cluster we will use the `swarm` container to send commands to the discovery backend hosted on Docker Hub. This discovery backend is only for low-volume development and testing purposes, not for production. Later on, when you run the Swarm manager and nodes, they register with the discovery backend as members of the cluster that's associated with the unique token. The discovery backend maintains an up-to-date list of cluster members and shares that list with the Swarm manager. The Swarm manager uses this list to assign tasks to the nodes.

1. Create a new container host, let's call it `token`:

        $ docker-machine create -d virtualbox token

2. Connect the Docker Client on your computer to the Docker Engine running on `token`.

        $ eval $(docker-machine env token)

    The client will send the `docker` commands in the following steps to the Docker Engine on `token`.

3.  Create a unique id for the Swarm cluster.

        $ docker run --rm swarm create
        .
        .
        .
        Status: Downloaded newer image for swarm:latest
        0ac50ef75c9739f5bfeeaf00503d4e6e

    The `docker run` command gets the latest `swarm` image and runs it as a container. The `create` argument makes the Swarm container connect to the Docker Hub discovery service and get a unique Swarm ID, also known as a "discovery token". The token appears in the output, it is not saved to a file on the host. The `--rm` option automatically removes the container when it exits.

    The discovery service keeps unused tokens for approximately one week.

4. Copy the discovery token from the last line of the previous output to a safe place. You can now remove the `token` host by running

        $ docker-machine rm token

## Create the Swarm manager and nodes

You will now use Docker Machine to provision three VMs running Docker Engine and use the "discovery token" from the previous step to make them all part of the same cluster.

1. Create and run a VM named `manager`.

		$ docker-machine create -d virtualbox --swarm --swarm-master --swarm-discovery token://YOURTOKENHERE manager

4. Create and run a VM named `agent1`.

		$ docker-machine create -d virtualbox --swarm --swarm-discovery token://YOURTOKENHERE agent1

5. Create and run a VM named `agent2`.

		$ docker-machine create -d virtualbox --swarm --swarm-discovery token://YOURTOKENHERE agent2

> Troubleshooting: If your computer or hosts cannot reach Docker Hub, the
`docker-machine` or `docker run` commands that pull images may fail. In that
case, check the [Docker Hub status page](http://status.docker.com/) for
service availability. Then, check whether your computer is connected to the Internet.  Finally, check whether VirtualBox's network settings allow your hosts to connect to the Internet.

## Manage your Swarm

Here, you connect to the cluster and review information about the Swarm manager and nodes. You tell the Swarm to run a container and check which node did the work.

1. Connect the Docker Client to the Swarm manager by running the following:

        $ eval $(docker-machine env --swarm manager)

    Because Docker Swarm uses the standard Docker API, you can connect to it using Docker Client and other tools such as Docker Compose, Dokku, Jenkins, and Krane, among others.

2. Get information about the Swarm.

        $ docker info

    As you can see, the output displays information about the two agent nodes and the one manager node in the Swarm.

3. Check the images currently running on your Swarm.

        $ docker ps

4. Run a container on the Swarm.

        $ docker run hello-world
        Hello from Docker.
        .
        .
        .

5. Use the `docker ps` command to find out which node the container ran on. For example:

        $ docker ps -a
        CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS                      PORTS               NAMES
        0b0628349187        hello-world         "/hello"                 20 minutes ago      Exited (0) 11 seconds ago                       agent1
        .
        .
        .

    In this case, the Swarm ran `hello-world` on `agent1`.

    By default, Docker Swarm uses the "spread" strategy to choose which node runs a container. When you run multiple containers, the spread strategy assigns each container to the node with the fewest containers.

## Where to go next

At this point, you've done the following:

 - Created a Swarm discovery token.
 - Created Swarm nodes using Docker Machine.
 - Managed a Swarm cluster and run containers on it.
 - Learned Swarm-related concepts and terminology.

However, Docker Swarm has many other aspects and capabilities.
For more information, visit [the Swarm landing page](https://www.docker.com/docker-swarm) or read the [Swarm documentation](https://docs.docker.com/swarm/).
