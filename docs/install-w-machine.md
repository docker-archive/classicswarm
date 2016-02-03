<!--[metadata]>
+++
title = "Evaluate Swarm in a sandbox"
description = "Evaluate Swarm in a sandbox"
keywords = ["docker, swarm, clustering, discovery, examples"]
[menu.main]
parent="workw_swarm"
+++
<![end-metadata]-->

# Evaluate Swarm in a sandbox

This getting started example shows you how to create a Docker Swarm, the
native clustering tool for Docker.

When you finish, you'll have a Docker Swarm up and running in VirtualBox on your
local Mac or Windows computer. You can use this Swarm as personal development
sandbox.

To use Docker Swarm on Linux, see [Create a Swarm for development](install-manual).

## Install Docker Toolbox

Download and install [Docker Toolbox](https://www.docker.com/docker-toolbox).

The toolbox installs a handful of tools on your local Windows or Mac OS X computer. In this exercise, you use three of those tools:

 - Docker Machine: To deploy virtual machines that run Docker Engine.
 - VirtualBox: To host the virtual machines deployed from Docker Machine.
 - Docker Client: To connect from your local computer to the Docker Engines on the VMs and issue docker commands to create the Swarm.

The following sections provide more information on each of these tools. The rest of the document uses the abbreviation, VM, for virtual machine.

## Create three VMs running Docker Engine

1. Open a terminal on your computer. Use Docker Machine to list any VMs in VirtualBox.

		$ docker-machine ls
		NAME         ACTIVE   DRIVER       STATE     URL                         SWARM
		default    *        virtualbox   Running   tcp://192.168.99.100:2376   

2. Optional: To conserve system resources, stop any virtual machines you are not using. For example, to stop the VM named `default`, enter:

		$ docker-machine stop default

3. Create and run a VM named `manager`.  

		$ docker-machine create -d virtualbox manager

3. Create and run a VM named `agent1`.  

		$ docker-machine create -d virtualbox agent1

3. Create and run a VM named `agent2`.  

		$ docker-machine create -d virtualbox agent2

Each command checks for a local copy of the latest VM image called boot2docker.iso. If the latest copy isn't available, Docker Machine downloads the latest image from Docker Hub. Then, Docker Machine uses boot2docker.iso to create a VM that automatically runs Docker Engine.

## Create a Swarm discovery token

1. Connect the Docker Client on your computer to the Docker Engine running on `manager`.

        $ eval $(docker-machine env manager)

    The client will send the `docker` commands in the following steps to the Docker Engine on on `manager`.

2.  Create a unique id for the Swarm cluster.

        $ docker run --rm swarm create
        .
        .
        .
        Status: Downloaded newer image for swarm:latest
        f6dc2febe7d321b987d30228e8c7a21b

    The `docker run` command gets the latest `swarm` image and runs it as a container. The `create` argument makes the Swarm container connect to the Docker Hub discovery service and get a unique Swarm ID, also known as a "discovery token". The `--rm` option automatically cleans up the container and removes the file system when the container exits.

    The discovery service keeps unused tokens for approximately one week.

3. Copy the discovery token from the last line of the previous output to a safe place.

## Create three Swarm nodes

1. Get the IP addresses of the three VMs.

        docker-machine ls

2. Use the following syntax to run a Swarm container as the primary Swarm manager on `manager`.

        docker run -t -p <your_selected_port>:2375 -t swarm manage token://<cluster_id> >

    For example:

        docker run -t -p 2370:2375 -t swarm manage token://f6dc2febe7d321b987d30228e8c7a21b

3. Connect Docker Client to `agent1`.

        eval $(docker-machine env agent1)

4. Use the following syntax to run a Swarm container as an agent on `agent1`.

        docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>

    For example:

        docker run -d swarm join --addr=192.168.99.101:2375 token://f6dc2febe7d321b987d30228e8c7a21b

5. Connect Docker Client to `agent2`.

        eval $(docker-machine env agent2)

6.  Run a Swarm container as an agent on `agent2`.

        docker run -d swarm join --addr=192.168.99.102:2376 token://f6dc2febe7d321b987d30228e8c7a21b

## Manage your Swarm

1. Connect the Docker Client to the Swarm.

        $ eval $(docker-machine env swarm)

    Because Docker Swarm uses the standard Docker API, you can connect to it using  Docker Client and other tools such as Docker Compose, Dokku, Jenkins, and Krane, among others.  

2. Get information about the Swarm.

        $ docker info

    As you can see, the output displays information about the two agent nodes and the one master node in the Swarm.

3. Check the images currently running on your Swarm.

        $ docker ps

4. Run a container on the Swarm.

        $ docker run hello-world
        Hello from Docker.
        .
        .
        .

5. Use the `docker ps` command to find out which node the container ran on.

        $ docker ps  -a
        CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS                      PORTS               NAMES
        0b0628349187        hello-world         "/hello"                 20 minutes ago      Exited (0) 20 minutes ago                       agent1
        .
        .
        .

    In this case, the Swarm ran 'hello-world' on the 'swarm1'.

    By default, Docker Swarm uses the "spread" strategy to choose which node runs a container. When you run multiple containers, the spread strategy assigns each container to the node with the fewest containers.

## Where to go next

At this point, you've done the following:
 - Created a Swarm discovery token.
 - Created Swarm nodes using Docker Machine.
 - Managed a Swarm and run containers on it.
 - Learned Swarm-related concepts and terminology.

However, Docker Swarm has many other aspects and capabilities.
For more information, visit [the Swarm landing page](https://www.docker.com/docker-swarm) or read the [Swarm documentation](https://docs.docker.com/swarm/).
