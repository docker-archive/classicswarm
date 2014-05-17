# Libswarm

## A library to orchestrate heterogeneous distributed systems

*libswarm* is a lightweight, standardized abstraction for federating and orchestrating
distributed systems. It can discover and communicate with all machines in your system,
using a variety of backend adaptors, and expose them via a single, straightforward API.

This is very useful if your system spans multiple locations on the Internet (your home
network, office lan, colocated rack, cloud provider, etc.) and uses heterogeneous tools
for deployment, clustering and service discovery (manually configured IPs on your laptop,
zeroconf at the office, ssh tunnels + custom DNS on the rack, Mesos, Consul or Etcd/Fleet
on your cloud provider, etc.).

*libswarm* federates all these elements under a unified namespace and lets you orchestrate
them using a lightweight and flexible API.

Because the *libswarm* API is dead-simple to learn and built on standard components, there
is a growing ecosystem of tools already compatible with it, which you can use to manage
your swarm out of the box:

* `swarmctl` is a simple command-line utility which lets you easily compose any combination
of backends, and orchestrate them straight from the terminal.

* (coming soon) *Shipyard* offers a powerful web interface to manage any swarm-compatible
distributed system straight from the browser.

* (coming soon) *Swarmlog* is a logging tool which consumes all log streams from your swarm
and forwards them to a variety of logging backends.

* (coming soon) *Deis*, the popular Docker-based application platform, supports *libswarm*
natively, and can deploy applications to it out-of-the-box.

* (coming soon) *Drone.io*, the Docker-based continuous integration platform, can spawn
build and test workers on any swarm.

* (coming soon) *Fig*, the popular development tool, supports libswarm out of the box.

* (coming soon) *Docker*, the open-source container engine, has announced that they will
adopt *libswarm* as a built-in, and recommend it as the preferred way to orchestrate
and manage a Docker-based cluster.


## Testing libswarm with swarmd

Libswarm ships with a simple daemon which can control all machines in your distributed
system using a variety of backend adaptors, and exposes it on a single, unified endpoint.

Currently swarmd uses the standard Docker API as its frontend, which means any tool which speaks
Docker can control swarmd transparently: dokku, flynn, deis, docker-ui, shipyard, fleet,
mesos... and of course the Docker client itself.

*Note: in the future swarmd will expose the Docker remote API as "just another backend", and
expose its own native API as a frontend.*

Usage example:

```
./swarmd tcp://localhost:4242 &
docker -H tcp://localhost:4242 info
```

You can listen on several frontend addresses, using any address format supported by Docker.
For example:

```
./swarmd tcp://localhost:4242 tcp://0.0.0.0:1234 unix:///var/run/docker.sock
```

## Backends

Libswarm supports the following backends:

### Debug backend

The debug backend simply catches all commands and prints them on the terminal for inspection.
It returns `StatusOK` for all commands, and never sends additional data. Therefore, docker
clients which expect data will fail to parse some of the responses.

Usage example:

```
./swarmd --backend debug tcp://localhost:4242
```

### Simulator backend

The simulator backend simulates a docker daemon with fake in-memory data. 
The state of the simulator persists across commands, so it's useful to analyse
the side-effects of commands, or for mocking and testing.

Currently the simulator only implements the `containers` command. It can be passed
arguments at load-time, and will use these arguments as a list of containers.

For example:

```
./swarmd --backend 'simulator container1 container2 container3' tcp://localhost:4242 &
docker -H tcp://localhost:4242 ps
```

In this example the docker client should report 3 containers: `container1`, `container2` and `container3`.

### Forward backend

The forward backend connects to a remote Docker API endpoint. It then forwards
all commands it receives to that remote endpoint, and forwards all responses
back to the frontend.

For example:

```
./swarmd --backend 'simulator myapp' unix://a.sock &
./swarmd --backend 'forward unix://a.sock' unix://b.sock
docker -H unix://b.sock ps
```

This last command should report 1 container: `myapp`.


## Creators

**Solomon Hykes**

- <http://twitter.com/solomonstre>
- <http://github.com/shykes>

## Copyright and license

Code and documentation copyright 2013-2014 Docker, inc. Code released under the Apache 2.0 license.
Docs released under Creative commons.
