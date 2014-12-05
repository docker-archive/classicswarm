## Swarm: a Docker-native clustering system [![Build Status](https://travis-ci.org/docker/swarm.svg?branch=master)](https://travis-ci.org/docker/swarm)

![Docker Swarm Logo](logo.png?raw=true "Docker Swarm Logo")

`swarm` is a simple tool which controls a cluster of Docker hosts and exposes it as a single "virtual" host.

`swarm` uses the standard Docker API as its frontend, which means any tool which speaks Docker can control swarm transparently: dokku, fig, krane, flynn, deis, docker-ui, shipyard, drone.io, Jenkins... and of course the Docker client itself.

Like the other Docker projects, `swarm` follows the "batteries included but removable" principle. It ships with a simple scheduling backend out of the box. The goal is to provide a smooth out-of-box experience for simple use cases, and allow swapping in more powerful backends, like `Mesos`, for large scale production deployments.

### Installation

######1 - Download the current source code.
```sh
go get github.com/docker/swarm
```

######2 - Compile and install `swarm`
```sh
go install github.com/docker/swarm
```

######3 - Nodes setup
The only requirement for Swarm nodes is to run a regular Docker daemon.

In order for Swarm to be able to communicate with its nodes, they must bind on a network interface.
This can be achieved by starting Docker with the `-H` flag (e.g. `-H 0.0.0.0:2375`).

Currently, nodes must be running the Docker **master** version.
Master binaries are available here: https://master.dockerproject.com/

### Example usage

```bash
# create a cluster
$ swarm create
6856663cdefdec325839a4b7e1de38e8

# on each of your nodes, start the swarm agent
$ swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip1:2375>
$ swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip2:2375>
$ swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip3:2375>
...

# start the manager on any machine or your laptop
$ swarm manage --token=6856663cdefdec325839a4b7e1de38e8 --addr=<swarm_ip:2375>

# use the regular docker cli
$ docker -H <swarm_ip:2375> info
$ docker -H <swarm_ip:2375> run ... 
$ docker -H <swarm_ip:2375> ps 
$ docker -H <swarm_ip:2375> logs ...
...

# list nodes in your cluster
$ swarm list --token=6856663cdefdec325839a4b7e1de38e8
http://<docker_daemon_ip1:2375>
http://<docker_daemon_ip2:2375>
http://<docker_daemon_ip3:2375>
```

## Participating

We welcome pull requests and patches; come say hi on IRC, #docker-swarm on freenode.

## Creators

**Andrea Luzzardi**

- <http://twitter.com/aluzzardi>
- <http://github.com/aluzzardi>

**Victor Vieux**

- <http://twitter.com/vieux>
- <http://github.com/vieux>

## Copyright and license

Code and documentation copyright 2014 Docker, inc. Code released under the Apache 2.0 license.
Docs released under Creative commons.

