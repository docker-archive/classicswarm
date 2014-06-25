# Libswarm

*libswarm* is a toolkit for composing network services.

It defines a standard interface for services in a distributed system to communicate with each other. This lets you:

1. Compose complex architectures from reusable building blocks
2. Avoid vendor lock-in by swapping any service out with another

An extensive library of services is included, and you can also write your own using a simple API.

Here are some examples of what you can do with libswarm:

* Aggregate all your Docker containers across multiple hosts and infrastructure providers, as if they were running on a single host.

* Bridge your in-house DNS-based service discovery with that new shiny Consul deployment, without getting locked into either.

* Swap in a new clustering and service discovery system, without changing any application code.

* Collect logs across an in-house Mesos cluster, a Cloudfoundry deployment and individual servers staggered in 3 different datacenters, forward them to your legacy syslog deployment, then perform custom analytics on them.

* Simulate your entire service topology in a single process, then scale it out simply by re-arranging adapters.

* Organize your application as loosely coupled services from day 1, without over-engineering.
 
## Installation

First get the go dependencies:

```sh
go get github.com/docker/libswarm/...
```

Then you can compile `swarmd` with:

```sh
go install github.com/docker/libswarm/swarmd
```

If `$GOPATH/bin` is in your `PATH`, you can invoke `swarmd` from the CLI.

```sh
$ swarmd -h
NAME:
   swarmd - Compose distributed systems from lightweight services

USAGE:
   swarmd [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --version, -v	print the version
   --help, -h		show help
```

## Built-in services

### Docker server

*Maintainer: Ben Firshman*

This service runs a Docker remote API server, allowing the Docker client and
other Docker tools to control libswarm services. With no arguments, it listens
on port `4243`, but you can specify any port you like using `tcp://0.0.0.0:9999`,
`unix:///tmp/docker` etc.


### Docker client

*Maintainer: Aanand Prasad*

This service can be used to control a Docker Engine from libswarm services. It takes one argument, the Docker host to connect to. For example: `dockerclient tcp://10.1.2.3:4243`

### SSH tunnel

*Help wanted!*

### Etcd

*Help wanted!*

### Geard

*Clayton Coleman*

### Fork-exec

*Solomon Hykes*

### Mesos

*Help wanted!*

### Shipyard

*Brian Goff*

### Fleet

*Help wanted!*

### Google Compute

*Brendan Burns*

### Rackspace Cloud

The Rackspace Cloud adapter allows users to forward commands to Docker
instances deployed on the Rackspace Cloud.

Usage of the adapter begins by setting the **--auth-user**, **--region**
switches. In addition, the adapter expects the environment variable
**OS_API_KEY** to be set to the API key of the user specified using the
**--auth-user** switch.

The adapter makes use of SSH to create secure tunnels to Rackspace Cloud
hosts. Please make sure that the private key specified using the **--key**
switch is in the hosts' ```authorized_keys``` for the user specified
using the **--user** switch.

The Rackspace Cloud adapter will attempt to build hosts as needed. Because
of this, the **--flavor** and **--image** switches are required. The image
specified must host a Docker daemon that auto-starts and binds to
tcp://127.0.0.1 on the port specified using the **--tun-port** switch.

#### Example

The following command sets up the rax adapter to target the Rackspace DFW
region for buildout.


```
export OS_API_KEY=123456789abcdef
rax --auth-user test --region dfw --key /etc/swarmd/rax_key --user test --flavor 3 --image 4c0fd362-1083-4f0e-8b39-e17bf00a2da5
```

Maintainer: [*John Hopper*](mailto:john.hopper@jpserver.net)

### Orchard

*Maintainer: Aanand Prasad*

Control an [Orchard](https://www.orchardup.com/) host from libswarm. It takes two arguments, an Orchard API token and the name of the Orchard host to control.

### Amazon EC2

*Help wanted!*

### Consul

*Help wanted!*

### OpenStack Nova

*Help wanted!*

### Digital Ocean

*Help wanted!*

### SoftLayer

*Help wanted!*

### ZeroRPC

*Help wanted!*

### Debug

The debug service simply catches all messages and prints them on the terminal for inspection.


## Testing libswarm with swarmd

Libswarm ships with a simple daemon which can control services in your distributed system.

Usage example:


Run swarmd without arguments to list available services:

```
./swarmd
```

Pass a service name as argument to load it:

```
./swarmd fakeclient
```

You can pass arguments to the service, like a shell command:

```
./swarmd 'dockerserver tcp://localhost:4243'
```

You can call multiple services. They will be executed in parallel, with the output
of each backend connected to the input of the next, just like unix pipelines.

This allows for very powerful composition.

```
./swarmd 'dockerserver tcp://localhost:4243' 'debug' 'dockerclient unix:///var/run/docker.sock'
```

## Creators

**Solomon Hykes**

- <http://twitter.com/solomonstre>
- <http://github.com/shykes>

## Copyright and license

Code and documentation copyright 2013-2014 Docker, inc. Code released under the Apache 2.0 license.
Docs released under Creative commons.
