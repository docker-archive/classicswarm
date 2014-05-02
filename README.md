# Swarmd

## Control a distributed system with the Docker API

`swarmd` is a simple daemon which can control all machines in your distributed
system using a variety of backend adaptors, and exposes it on a single, unified endpoint.

swarmd uses the standard Docker API as its frontend, which means any tool which speaks
Docker can control swarmd transparently: dokku, flynn, deis, docker-ui, shipyard, fleet,
mesos... and of course the Docker client itself.

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

`Swarmd` supports the following backends:

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


