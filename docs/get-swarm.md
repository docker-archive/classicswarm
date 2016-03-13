<!--[metadata]>
+++
title = "How to get Swarm"
description = "Running a Swarm container on Docker Engine. Run a Swarm binary on the host OS without Docker Engine."
keywords = ["docker, Swarm, container, binary, clustering, install, installation"]
[menu.main]
identifier="how-to-get-and-run-Swarm"
parent="workw_swarm"
weight=-90
+++
<![end-metadata]-->

# How to get Docker Swarm

You can create a Docker Swarm cluster using the `swarm` executable image from a
container or using an executable `swarm` binary you install on your system. This
page introduces the two methods and discusses their pros and cons.

## Create a cluster with an interactive container

You can use the Docker Swarm official image to create a cluster. The image is
built by Docker and updated regularly through an automated build. To use the
image, you run it a container via the Engine `docker run` command. The image has
multiple options and subcommands you can use to create and manage a Swarm cluster.

The first time you use any image, Docker Engine checks to see if you already have the image in your environment. By default Docker runs the `swarm:latest` version but you can also specify a tag other than `latest`. If you have an image locally but a newer one exists on Docker Hub, Engine downloads it.

### Run the Swarm image from a container

1. Open a terminal on a host running Engine.

    If you are using Mac or Windows, then you must make sure you have started an Docker Engine host running and pointed your terminal environment to it with the Docker Machine commands. If you aren't sure, you can verify:

        $ docker-machine ls
        NAME      ACTIVE   URL          STATE     URL                         SWARM   DOCKER    ERRORS
        default   *       virtualbox   Running   tcp://192.168.99.100:2376           v1.9.1    

    This shows an environment running an Engine host on the `default` instance.

2. Use the `swarm` image to execute a command.

    The easiest command is to get the help for the image. This command shows all the options that are available with the image.

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

    In this example, the `swarm` image did not exist on the Engine host, so the
    Engine downloaded it. After it downloaded, the image executed the `help`
    subcommand to display the help text. After displaying the help, the `swarm`
    image exits and returns you to your terminal command line.

3. List the running containers on your Engine host.

        $ docker ps
        CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES

    Swarm is no longer running. The `swarm` image exits after you issue it a command.

### Why use the image?

Using a Swarm container has three key benefits over other methods:

* You don't need to install a binary on the system to use the image.
* The single command `docker run` command gets and run the most recent version of the image every time.
* The container isolates Swarm from your host environment. You don't need to perform or maintain shell paths and environments.

Running the Swarm image is the recommended way to create and manage your Swarm cluster. All of Docker's documentation and tutorials use this method.

## Run a Swarm binary

Before you run a Swarm binary directly on a host operating system (OS), you compile the binary from the source code or get a trusted copy from another location. Then you run the Swarm binary.

To compile Swarm from source code, refer to the instructions in
[CONTRIBUTING.md](http://github.com/docker/swarm/blob/master/CONTRIBUTING.md).


### Why use the binary?

Using a Swarm binary this way has one key benefit over other methods: If you are
a developer who contributes to the Swarm project, you can test your code changes
without "containerizing" the binary before you run it.

Running a Swarm binary on the host OS has disadvantages:

* Compilation from source is a burden.
* The binary doesn't have the benefits that
Docker containers provide, such as isolation.
* Most Docker documentation and tutorials don't show this method of running swarm.

Lastly, because the Swarm nodes don't use Engine, you can't use Docker-based
software tools, such as Docker Engine CLI at the node level.

## Related information

* [Docker Swarm official image](https://hub.docker.com/_/swarm/) repository on Docker Hub
* [Provision a Swarm with Docker Machine](provision-with-machine.md)
