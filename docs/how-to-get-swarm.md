<!--[metadata]>
+++
title = "Get the Swarm software"
description = "Get the Swarm software"
keywords = ["docker, Swarm, clustering, image, binary"]
[menu.main]
parent="workw_swarm"
weight=-97
+++
<![end-metadata]-->

# Get and run the Swarm software

You can create a Docker Swarm cluster using the `swarm` executable image from a  container or using an executable `swarm` binary you build and install on your system. This page introduces the two methods and discusses their pros and cons

## Create a cluster with an interactive container

You can use the Docker Swarm image to create and manage a Swarm cluster. The image is built by Docker and updated through an automated build. To use the image, you use the Engine `docker run` to launch the image in a container. The image has multiple subcommands you can use.

The first time you use the image, Docker Engine checks to see if you already have the image in your environment. By default Docker runs the `swarm:latest` version but you can also specify a tag other than `latest`. If you have an image but a newer one exists on Docker Hub, Engine downloads it.

### Run the Swarm image from a container

1. Open a terminal on a host running Docker Engine.

    If you are using Mac or Windows, make sure the terminal environment is set.

2. Use the Swarm image to list its help.

        $ docker run swarm --help
        Unable to find image 'swarm:latest' locally
        latest: Pulling from library/swarm
        d681c900c6e3: Pull complete
        188de6f24f3f: Pull complete
        90b2ffb8d338: Pull complete
        237af4efea94: Pull complete
        3b3fc6f62107: Pull complete
        7e6c9135b308: Pull complete
        986340ab62f0: Pull complete
        a9975e2cc0a3: Pull complete
        Digest: sha256:c21fd414b0488637b1f05f13a59b032a3f9da5d818d31da1a4ca98a84c0c781b
        Status: Downloaded newer image for swarm:latest
        Usage: swarm [OPTIONS] COMMAND [arg...]

        A Docker-native clustering system

        Version: 1.0.1 (744e3a3)

        Options:
          --debug			debug mode [$DEBUG]
          --log-level, -l "info"	Log level (options: debug, info, warn, error, fatal, panic)
          --help, -h			show help
          --version, -v			print the version

        Commands:
          create, c	Create a cluster
          list, l	List nodes in a cluster
          manage, m	Manage a docker cluster
          join, j	join a docker cluster
          help, h	Shows a list of commands or help for one command

        Run 'swarm COMMAND --help' for more information on a command.

    In this example, the image did not exist on the sever, so the Engine downloaded it.  After it downloaded, the `swarm` image executed the and displayed the help. After displaying the help, the image exited.

3. List the running containers.

        $ docker ps
        CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES

    Swarm is no longer running. The `swarm` image exits after you issue it a command.

### Why use the image?

Using a Swarm container has three key benefits over other methods:

* You don't need to install a binary on the system to use it.
* The single command `docker run` command gets and run the most recent version of the image.
* The container isolates Swarm from XXX? This is good because XXX?

Running the Swarm image is the recommended way to create and manage your Swarm cluster. All of Docker's documentation and tutorials use this method.

## Build and run a Swarm binary

Before you run a Swarm binary directly on a host operating system (OS), you
compile the binary from the source code or get a trusted copy from another
location. Then, you run the Swarm binary.

The Docker Swarm repository on GitHub contains [instructions for building the binary](https://github.com/docker/swarm/blob/master/CONTRIBUTING.md).

Building the binary is recommend if you are a developer who contributes to the
Swarm project, you can test your code changes without "containerizing" the
binary before you run it.

Running a Swarm binary on the host OS has definite disadvantages. Getting and
running the latest version of Swarm is a burden. The binary doesn't get the
benefits that Docker containers provide, such as isolation. Most documentation
and tutorials aren't based on this method of running Swarm.

Lastly, when you use this binary, the Swarm nodes don't use Engine, you can't
use Docker-based software tools, such as Docker CLI client TO Orchestrate at the
node level.

## Related information

* [Evaluate Swarm in a sandbox](install-w-machine.md)
* [Build a Swarm cluster for production](install-on-aws.md)
* [Docker Swarm official image](https://hub.docker.com/_/swarm/)
