<!--[metadata]>
+++
title = "How to get and run Docker Swarm"
description = "Running a Swarm container on Docker Engine. Run a Swarm binary on the host OS without Docker Engine."
keywords = ["docker, Swarm, container, binary, clustering"]
[menu.main]
identifier="how-to-get-and-run-Swarm"
parent="mn_install"
weight=9
+++
<![end-metadata]-->

# Get and run Swarm

You can create an instance of Docker Swarm using a *container* on Docker Engine, or using an *executable* binary on your system. This page covers both ways of doing it.

## Using a container to create a Swarm instance

<Rolfe - Insert diagram of container, DE, and host computer.>

You can use the [Docker Swarm official image](https://hub.docker.com/_/swarm/) to create a cluster. Docker automatically builds and updates this image. To create a container from the image, you use the Docker Engine command: `docker run`. The image has multiple subcommands you can use.

The first time you use the image, the Engine checks to see if you already have the image in your environment. By default, Docker runs the `swarm:latest` version, but you can specify a tag other than `latest`. If you have an image but a newer one exists on Docker Hub, Engine downloads it.

### Run Swarm using a container

1. Open a terminal on a host running Docker Engine.

    If you are using Mac or Windows, set the terminal environment to a VM that's running docker: `eval $(docker-machine env my-vm-name)`.

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

    In this example, the image did not exist on the server, so Engine downloaded it. Then, Engine used the image to create and run Swarm inside a container.     The `--rm` option automatically cleans up the container and removes the file system when the container exits. After running the help command, Swarm exited, and Engine stopped the container.

3. List the running containers.

        $ docker ps
        CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES

    The Swarm container doesn't show up in the list because Engine has stopped it.

### Why use a container?

Using a Swarm container has three key benefits over other methods:

* You don't need to install a binary on the system to use it.
* The single command `docker run` command gets and runs the most recent version of the image.
* The container isolates and protects Swarm from other apps running on the system.

Running the Swarm image is the recommended way to create and manage your Swarm cluster. All of Docker's documentation and tutorials use this method.

## Using a binary to create Swarm instance

<Rolfe - Insert diagram of binary on host computer.>

Before you run a Swarm binary directly on a host operating system (OS), you compile the binary from the source code or get a trusted copy from another location. Then you run the Swarm binary.

### Build and use the binary

Beginning with Swarm 0.4 golang 1.4.x or later is required for building Swarm.
Refer to the [Go installation page](https://golang.org/doc/install#install)
to download and install the golang 1.4.x or later package.
> **Note**: On Ubuntu 14.04, the `apt-get` repositories install golang 1.2.1 version by
> default. So, do not use `apt-get` but install golang 1.4.x manually using the
> instructions provided on the Go site.

1. Install [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

    The source code is also available from the releases page on GitHub. You should link to it.

2. Install [godep](https://github.com/tools/godep).

    > **Note** `GOPATH` should be set while installing godep.

3. Install the `swarm` binary in the `$GOPATH/bin` directory. An easy way to do this is using the `go get` command.

        $ go get github.com/docker/swarm

    You can also do this manually using the following commands:

        $ mkdir -p $GOPATH/src/github.com/docker/
        $ cd $GOPATH/src/github.com/docker/
        $ git clone https://github.com/docker/swarm
        $ cd swarm
        $ $GOPATH/bin/godep go install .

    Then you can find the swarm binary under `$GOPATH/bin`.

4. Run the help command

        $ swarm --help


### Why use a binary?

Using a Swarm binary this way has one key benefit over other methods: If you are a developer who contributes to the Swarm project, you can test your code changes without "containerizing" the binary before you run it.

Running a Swarm binary on the host OS has disadvantages: Getting and running the latest version of Swarm is a burden. The binary doesn't get the benefits that Docker containers provide, such as isolation. Most documentation and tutorials aren't based on this method of running Swarm. Lastly, because the swarm nodes don't use Docker Engine, you can't use Docker-based software tools, such as Docker Client an Orchestrate at the node level.
