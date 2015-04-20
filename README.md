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

## Installation and documentation

Full documentation [is available here](http://docs.docker.com/swarm/).

## Development installation

You can download and install from source instead of using the Docker
image. Ensure you have golang, godep and the git client installed.

**For example, on Ubuntu you'd run:**

```bash
$ apt-get install golang git
$ go get github.com/tools/godep
```

You may need to set `$GOPATH`, e.g `mkdir ~/gocode; export GOPATH=~/gocode`.

**For example, on Mac OS X you'd run:**

```bash
$ brew install go
$ export GOPATH=~/go
$ export PATH=$PATH:~/go/bin
$ go get github.com/tools/godep
```

Then install the `swarm` binary:

```bash
$ mkdir -p $GOPATH/src/github.com/docker/
$ cd $GOPATH/src/github.com/docker/
$ git clone https://github.com/docker/swarm
$ cd swarm
$ godep go install .
```

From here, you can follow the instructions [in the main documentation](http://docs.docker.com/swarm/),
replacing `docker run swarm` with just `swarm`.

## Participating

We welcome pull requests and patches; come say hi on IRC, #docker-swarm on freenode.

Our planning process and release cycle are detailed on the [wiki](https://github.com/docker/swarm/wiki)

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
