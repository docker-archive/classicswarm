# Swarm: a Docker-native clustering system

[![GoDoc](https://godoc.org/github.com/docker/swarm?status.png)](https://godoc.org/github.com/docker/swarm)
[![Build Status](https://travis-ci.org/docker/swarm.svg?branch=master)](https://travis-ci.org/docker/swarm)
[![Coverage Status](https://coveralls.io/repos/docker/swarm/badge.svg)](https://coveralls.io/r/docker/swarm)

![Docker Swarm Logo](logo.png?raw=true "Docker Swarm Logo")

Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual host.

Swarm serves the standard Docker API, so any tool which already communicates
with a Docker daemon can use Swarm to transparently scale to multiple hosts:
Dokku, Compose, Krane, Flynn, Deis, DockerUI, Shipyard, Drone, Jenkins... and,
of course, the Docker client itself.

Like other Docker projects, Swarm follows the "batteries included but removable"
principle. It ships with a set of simple scheduling backends out of the box, and as
initial development settles, an API will be developed to enable pluggable backends.
The goal is to provide a smooth out-of-the-box experience for simple use cases, and
allow swapping in more powerful backends, like Mesos, for large scale production
deployments.

## Installation for Swarm Users

For installing swarm for using in your environment, use the Docker Swarm
documentation on [docs.docker.com](http://docs.docker.com/swarm/).

## Installation for Swarm Developers

Developers should always download and install from source rather than
using the Docker image.

### Prerequisites

1. Beginning with Swarm 0.4 golang 1.4.x or later is required for building Swarm. 
Refer to the [Go installation page](https://golang.org/doc/install#install)
to download and install the golang 1.4.x or later package.
> **Note**: On Ubuntu 14.04, the `apt-get` repositories install golang 1.2.1 version by
> default. So, do not use `apt-get` but install golang 1.4.x manually using the
> instructions provided on the Go site.

2. Install [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

3. Install [godep](https://github.com/tools/godep).

### Clone and build Swarm

> **Note** `GOPATH` should be set when install godep in above step.

Install the `swarm` binary in the `$GOPATH/bin` directory. An easy way to do this 
is using the `go get` command.

```bash
$ go get github.com/docker/swarm
```

You can also do this manually using the following commands:

```bash
$ mkdir -p $GOPATH/src/github.com/docker/
$ cd $GOPATH/src/github.com/docker/
$ git clone https://github.com/docker/swarm
$ cd swarm
$ $GOPATH/bin/godep go install .
```

Then you can find the swarm binary under `$GOPATH/bin`.

From here, you can follow the instructions [in the main documentation](http://docs.docker.com/swarm/),
replacing `docker run swarm` with just `swarm`.

## Participating

You can contribute to Docker Swarm in several different ways:

  - If you have comments, questions, or want to use your knowledge to help others, come join the conversation on IRC. You can reach us at #docker-swarm on Freenode.

  - To report a problem or request a feature, please file an issue.

  - Of course, we welcome pull requests and patches. For information on making feature requests, follow the process suggested here.

Finally, if you want to see what we have for the future and learn more about our release cycles, all this information is detailed on the [wiki](https://github.com/docker/swarm/wiki)


## Creators

**Andrea Luzzardi**

- <http://twitter.com/aluzzardi>
- <http://github.com/aluzzardi>

**Victor Vieux**

- <http://twitter.com/vieux>
- <http://github.com/vieux>

## Copyright and license

Code and documentation copyright 2014-2015 Docker, inc. Code released under the
Apache 2.0 license.

Docs released under Creative commons.
