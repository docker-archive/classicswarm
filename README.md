# Swarm: a Docker-native clustering system [![Build Status](https://travis-ci.org/docker/swarm.svg?branch=master)](https://travis-ci.org/docker/swarm)

![Docker Swarm Logo](logo.png?raw=true "Docker Swarm Logo")

`swarm` is a simple tool which controls a cluster of Docker hosts and exposes it
as a single "virtual" host.

`swarm` uses the standard Docker API as its frontend, which means any tool which
speaks Docker can control swarm transparently: dokku, fig, krane, flynn, deis,
docker-ui, shipyard, drone.io, Jenkins... and of course the Docker client itself.

Like the other Docker projects, `swarm` follows the "batteries included but removable"
principle. It ships with a simple scheduling backend out of the box, and as initial
development settles, an API will develop to enable pluggable backends. The goal is
to provide a smooth out-of-box experience for simple use cases, and allow swapping
in more powerful backends, like `Mesos`, for large scale production deployments.

## Installation

###1 - Docker image.
The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

```sh
docker pull swarm
```

###2 - Alternative: Download and install from source.
Alternatively, you can download and install from source instead of using the
Docker image.

Ensure you have golang and git client installed (e.g. `apt-get install golang git` on Ubuntu).
You may need to set `$GOPATH`, e.g `mkdir ~/gocode; export GOPATH=~/gocode`.

The install `swarm` binary to your `$GOPATH` directory.

```bash
go get -u github.com/docker/swarm
```

###3 - Nodes setup
The only requirement for Swarm nodes is to run a regular Docker daemon (version
`1.4.0` and later).

In order for Swarm to be able to communicate with its nodes, they must bind on a
network interface. This can be achieved by starting Docker with the `-H` flag
(e.g. `-H tcp://0.0.0.0:2375`).

# Example usage

```bash
# create a cluster
$ docker run --rm swarm create
6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>

# on each of your nodes, start the swarm agent
#  <node_ip> doesn't have to be public (eg. 192.168.0.X),
#  as long as the swarm manager can access it.
$ docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>

# start the manager on any machine or your laptop
$ docker run -t -p 2375:<swarm_port> -t swarm manage token://<cluster_id>

# use the regular docker cli
$ docker -H tcp://<swarm_ip:swarm_port> info
$ docker -H tcp://<swarm_ip:swarm_port> run ...
$ docker -H tcp://<swarm_ip:swarm_port> ps
$ docker -H tcp://<swarm_ip:swarm_port> logs ...
...

# list nodes in your cluster
$ docker run --rm swarm list token://<cluster_id>
<node_ip:2375>
```

See [here](discovery) for more information about
other discovery services.

## Advanced Scheduling

See [filters](scheduler/filter) and [strategies](scheduler/strategy) to learn
more about advanced scheduling.

## TLS

Swarm supports TLS authentication between the CLI and Swarm but also between
Swarm and the Docker nodes.

In order to enable TLS, the same command line options as Docker can be specified:

`swarm manage --tlsverify --tlscacert=<CACERT> --tlscert=<CERT> --tlskey=<KEY> [...]`

Please refer to the [Docker documentation](https://docs.docker.com/articles/https/)
for more information on how to set up TLS authentication on Docker and generating
the certificates.

Note that Swarm certificates must be generated with`extendedKeyUsage = clientAuth,serverAuth`.

## Participating

We welcome pull requests and patches; come say hi on IRC, #swarm on freenode.

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
