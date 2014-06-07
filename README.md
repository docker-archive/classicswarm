# Libswarm

## A minimalist toolkit to compose network services


*libswarm* is a minimalist toolkit to compose network services.

It exposes a simple API for the following tasks:

* *Clustering*: deploy services on pools of interchangeable machines
* *Composition*: combine multiple services into higher-level services of arbitrary complexity - it's services all the way down!
* *Interconnection*: services can reliably and securely communicate with each other using asynchronous message passing, request/response, or raw sockets.
* *Scale* services can run concurrently in the same process using goroutines and channels; in separate processes on the same machines using high-performance IPC;
on multiple machines in a local network; or across multiple datacenters.
* *Integration*: incorporate your existing systems into your swarm. libswarm includes adapters to many popular infrastructure tools and services: docker, dns, mesos, etcd, fleet, deis, google compute, rackspace cloud, tutum, orchard, digital ocean, ssh, etc. It’s very easy to create your own adapter: just clone the repository at 


## Adapters

Libswarm supports the following adapters:

### Debug adapter

The debug backend simply catches all messages and prints them on the terminal for inspection.


### Docker server adapter

*Maintainer: Ben Firshman*

### Docker client adapter

*Maintainer: Aanand Prasad*

### Pipeline adapter

*Maintainer: Ben Firshman*

### Filter adapter

*Maintainer: Solomon Hykes*

### Handler adapter

*Maintainer: Solomon Hykes*

### Task adapter

*Maintainer: Solomon Hykes*

### Go channel adapter

*Maintainer: Solomon Hykes*

### CLI adapter

*Maintainer: Solomon Hykes*

### Unix socket adapter

*Maintainer: Solomon Hykes*

### TCP adapter (client and server)

*Help wanted!*

### TLS adapter (client and server)

*Help wanted!*

### HTTP2/SPDY adapter (client and server)

*Maintainer: Derek McGowan*

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

### Rackspace cloud adapter

*John Hopper*

### EC2 adapter

*Help wanted!*

### Consul adapter

*Help wanted!*

### Openstack Nova adapter

*Help wanted!*

### Digital Ocean adapter

*Help wanted!*

### Softlayer adapter

*Help wanted!*

### Zerorpc adapter

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
