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

You'll use Docker Toolbox to install Docker Machine and some other tools on your computer. Then you'll use Docker Machine to provision a set of Docker Engine hosts. Lastly, you'll use Docker client to connect to the hosts, where you'll create a discovery token, create a cluster of one Swarm manager and nodes, and manage the cluster.

When you finish, you'll have a Docker Swarm up and running in VirtualBox on your
local Mac or Windows computer. You can use this Swarm as personal development
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

		$ docker-machine create -d virtualbox agent1

5. Create and run a VM named `agent2`.

		$ docker-machine create -d virtualbox agent2


Each create command checks for a local copy of the *latest* VM image, called boot2docker.iso. If it isn't available, Docker Machine downloads the image from Docker Hub. Then, Docker Machine uses boot2docker.iso to create a VM that automatically runs Docker Engine.

> Troubleshooting: If your computer or hosts cannot reach Docker Hub, the
`docker-machine` or `docker run` commands that pull images may fail. In that
case, check the [Docker Hub status page](http://status.docker.com/) for
service availability. Then, check whether your computer is connected to the Internet.  Finally, check whether VirtualBox's network settings allow your hosts to connect to the Internet.

## Create a Swarm discovery token

Here you use the discovery backend hosted on Docker Hub to create a unique discovery token for your cluster. This discovery backend is only for low-volume development and testing purposes, not for production. Later on, when you run the Swarm manager and nodes, they register with the discovery backend as members of the cluster that's associated with the unique token. The discovery backend maintains an up-to-date list of cluster members and shares that list with the Swarm manager. The Swarm manager uses this list to assign tasks to the nodes.

1. Connect the Docker Client on your computer to the Docker Engine running on `manager`.

        $ eval $(docker-machine env manager)

    The client will send the `docker` commands in the following steps to the Docker Engine on `manager`.

2.  Create a unique id for the Swarm cluster.

        $ docker run --rm swarm create
        .
        .
        .
        Status: Downloaded newer image for swarm:latest
        0ac50ef75c9739f5bfeeaf00503d4e6e

    The `docker run` command gets the latest `swarm` image and runs it as a container. The `create` argument makes the Swarm container connect to the Docker Hub discovery service and get a unique Swarm ID, also known as a "discovery token". The token appears in the output, it is not saved to a file on the host. The `--rm` option automatically cleans up the container and removes the file system when the container exits.

    The discovery service keeps unused tokens for approximately one week.

3. Copy the discovery token from the last line of the previous output to a safe place.

## Create the Swarm manager and nodes

Here, you connect to each of the hosts and create a Swarm manager or node.

1. List the VMs to check that they're set up and running. For example:

        $ docker-machine ls
        NAME      ACTIVE   DRIVER       STATE     URL                         SWARM   DOCKER   ERRORS
        agent1    -        virtualbox   Running   tcp://192.168.99.102:2376           v1.9.1
        agent2    -        virtualbox   Running   tcp://192.168.99.103:2376           v1.9.1
        manager   *        virtualbox   Running   tcp://192.168.99.100:2376           v1.9.1


2. Your client should still be pointing to Docker Engine on `manager`. Use the following syntax to run a Swarm container that functions as the primary manager on `manager`.

        $ docker run -d -p <your_selected_port>:3376 -t -v /var/lib/boot2docker:/certs:ro swarm manage -H 0.0.0.0:3376 --tlsverify --tlscacert=/certs/ca.pem --tlscert=/certs/server.pem --tlskey=/certs/server-key.pem token://<cluster_id>

    For example:

        $ docker run -d -p 3376:3376 -t -v /var/lib/boot2docker:/certs:ro swarm manage -H 0.0.0.0:3376 --tlsverify --tlscacert=/certs/ca.pem --tlscert=/certs/server.pem --tlskey=/certs/server-key.pem token://0ac50ef75c9739f5bfeeaf00503d4e6e

    The `-p` option maps a port 3376 on the container to port 3376 on the host. The `-v` option mounts the directory containing TLS certificates (`/var/lib/boot2docker` for the `manager` VM) into the container running Swarm manager in read-only mode.

3. Connect Docker Client to `agent1`.

        $ eval $(docker-machine env agent1)

4. Use the following syntax to run a Swarm container that functions as an agent on `agent1`. Replace `<node_ip>` with the IP address of the VM from above, or use the `docker-machine ip` command.

        $ docker run -d swarm join --addr=<node_ip>:<node_port> token://<cluster_id>

    For example:

        $ docker run -d swarm join --addr=$(docker-machine ip agent1):2376 token://0ac50ef75c9739f5bfeeaf00503d4e6e

5. Connect Docker Client to `agent2`.

        $ eval $(docker-machine env agent2)

6.  Run a Swarm container as an agent on `agent2`. For example:

        $ docker run -d swarm join --addr=$(docker-machine ip agent2):2376 token://0ac50ef75c9739f5bfeeaf00503d4e6e

## Manage your Swarm

Here, you connect to the cluster and review information about the Swarm manager and nodes. You tell the Swarm to run a container and check which node did the work.

1. Connect the Docker Client to the Swarm by updating the `DOCKER_HOST` environment variable.

        $ DOCKER_HOST=<manager_ip>:<your_selected_port>

    We use the `docker-machine ip` command, and we selected port 3376 for the Swarm manager.

        $ DOCKER_HOST=$(docker-machine ip manager):3376

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
