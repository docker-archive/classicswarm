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


### Creating a new backend
Create a simple my-backend.go:
```
func MyBackend() engine.Installer {
	return &myBackend{}
}

func (f *computeEngineForward) Install(eng *engine.Engine) error {
	eng.Register("mybackend", func(job *engine.Job) engine.Status {
	job.Eng.Register("containers", func(job *engine.Job) engine.Status {
			log.Printf("%#v", *job)
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
```

Then edit backends.go, and add your backend:
```
...
	MyBackend().Install(back)
...
```

## Creators

**Solomon Hykes**

- <http://twitter.com/solomonstre>
- <http://github.com/shykes>

## Copyright and license

Code and documentation copyright 2013-2014 Docker, inc. Code released under the Apache 2.0 license.
Docs released under Creative commons.
