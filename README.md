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

This service runs a Docker remote API server, allowing the Docker client and other Docker tools to control libswarm services.


### Docker client

*Maintainer: Aanand Prasad*

This service can be used to control a Docker Engine from libswarm services. It takes one argument, the Docker host to connect to. For example: `dockerclient tcp://10.1.2.3:4243`

### SSH tunnel

*Help wanted!*

### Auth

This service can be used to autheticate or verify the identity of messages passed through libswarm. The service has several providers. One provider must be selected using the
**--provider** swtich.

Authenticators in the auth service are passed a string environment map as their argument. This map is derived from any message arguments that begin with **--auth**. Authentactors may optionally edit the environment by removing, changing or adding elements. These elements are rewritten into msg.Args after the authentactor returns. This allows for identity information to flow upstream while allowing for the removal of sensitive arguments.

Common arguments that are recognized by all auth providers:
* auth-user **string**: represents the user being authenticated.
* auth-key **string**: represents a user key.
* auth-password **string**: represents a user password.

**Auth Example**
```
swarmd "dockerserver" "injector --auth-dict @authdb.json --auth-user test --auth-password password" "auth --provider inline" "debug"
[DEBUG] Ls --test test
```


### Utils/Injector

This service can be used to inject arugments into a libchan message mid-stream. Arguments passed to the injector will be injected as read. Any argument that begins with the '@' character is treated as a file path, resolved and then read into the message's arguments.

**Injecting Arguments**
```
swarmd "dockerserver" "injector --test test" "debug"
[DEBUG] Ls --test test
```

**Injecting Files as Arguments**
```
swarmd "dockerserver" "injector --test @test.json" "debug"
[DEBUG] Ls --test {"test": "test"}
```

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

*John Hopper*

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
