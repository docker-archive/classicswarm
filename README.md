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

