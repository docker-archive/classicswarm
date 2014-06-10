# Libswarm

*libswarm* is a toolkit for composing network services.

At its core, it defines a standard way for services in a distributed system to communicate with each other. This lets you:

1. Use them as building blocks to compose complex architectures
2. Avoid vendor lock-in by swapping any service out with another

An extensive library of services is included, and you can also write your own using a simple API.

Here are some examples of what you can do with libswarm:

* Aggregate all your Docker containers across multiple hosts and infrastructure providers, as if they were running on a single host.

* Bridge your in-house DNS-based service discovery with that new shiny Consul deployment, without getting locked into either.

* Swap in a new clustering and service discovery system, without changing any application code.

* Collect logs across an in-house Mesos cluster, a Cloudfoundry deployment and individual servers staggered in 3 different datacenters, forward them to your legacy syslog deployment, then perform custom analytics on them.

* Simulate your entire service topology in a single process, then scale it out simply by re-arranging adapters.

* Organize your application as loosely coupled services from day 1, without over-engineering.

## Built-in services

Libswarm supports the following adapters:

### Debug adapter

The debug backend simply catches all messages and prints them on the terminal for inspection.


### Docker server adapter

*Maintainer: Ben Firshman*







### Etcd adapter

*Help wanted!*

### Geard adapter

*Clayton Coleman*

### Fork-exec adapter

*Solomon Hykes*

### Mesos adapter

*Help wanted!*

### Shipyard adapter

*Brian Goff*

### Fleet adapter

*Help wanted!*

### Google Compute adapter

*Brendan Burns*

### Rackspace Cloud adapter

*John Hopper*

### EC2 adapter

*Help wanted!*

### Consul adapter

*Help wanted!*

### OpenStack Nova adapter

*Help wanted!*

### Digital Ocean adapter

*Help wanted!*

### SoftLayer adapter

*Help wanted!*

### ZeroRPC adapter

*Help wanted!*


## Testing libswarm with swarmd

Libswarm ships with a simple daemon which can control all machines in your distributed
system using a variety of backend adaptors, and exposes it on a single, unified endpoint.

Usage example:


Run swarmd without arguments to list available backends:

```
./swarmd
```

Pass a backend name as argument to load it:

```
./swarmd fakeclient
```

You can pass arguments to the backend, like a shell command:

```
./swarmd 'dockerserver tcp://localhost:4243'
```

You can call multiple backends. They will be executed in parallel, with the output
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
