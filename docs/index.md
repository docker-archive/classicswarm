---
page_title: Docker Swarm
page_description: Swarm: a Docker-native clustering system
page_keywords: docker, swarm, clustering
---

# Docker Swarm

Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual host.

Swarm serves the standard Docker API, so any tool which already communicates
with a Docker daemon can use Swarm to transparently scale to multiple hosts:
Dokku, Compose, Krane, Deis, DockerUI, Shipyard, Drone, Jenkins... and,
of course, the Docker client itself.

Like other Docker projects, Swarm follows the "batteries included but swappable"
principle. It ships with a simple scheduling backend out of the box, and as
initial development settles, an API will develop to enable pluggable backends.
The goal is to provide a smooth out-of-box experience for simple use cases, and
allow swapping in more powerful backends, like Mesos, for large scale production
deployments.

> **Note**: Swarm is currently in beta, so things are likely to change. We
> don't recommend you use it in production yet.

## Release Notes

### Version 0.2.0 (April 16, 2015)

For complete information on this release, see the
[0.2.0 Milestone project page](https://github.com/docker/swarm/wiki/0.2.0-Milestone-Project-Page).
In addition to bug fixes and refinements, this release adds the following:

* Increased API coverage. Swarm now supports more of the Docker API. For
details, see
[the API README](https://github.com/docker/swarm/blob/master/api/README.md).

* Swarm Scheduler: Spread strategy. Swarm has a new default strategy for
ranking engines, called "spread". With this strategy, Swarm will optimize
for the engine with the fewest running containers. For details, see the
[scheduler strategy README](https://github.com/docker/swarm/blob/master/scheduler/strategy/README.md)

* Swarm Scheduler: Soft affinities. Soft affinities allow Swarm more flexibility
in applying rules and filters for engine selection. If the scheduler can't obey a
filtering rule, it will discard the filter and fall back to the assigned
strategy. For details, see the [scheduler filter README](https://github.com/docker/swarm/tree/master/scheduler/filter#soft-affinitiesconstraints).

* Better integration with Compose. Swarm will now schedule inter-dependent
containers on the same host. For details, see
[PR #972](https://github.com/docker/compose/pull/972).

## Pre-requisites for running Swarm

Each machine of your cluster must be running Docker 1.4.0 or later.

By default, Docker listens on `unix:///var/run/docker.sock` to allow only local connections.

To enable communication between Swarm and Docker running on different machines,
each Docker must be 
[configured] (https://docs.docker.com/articles/basics/#bind-docker-to-another-hostport-or-a-unix-socket)
to listen to a TCP port.

While each machine's IP need not be public, the Swarm manager must be able to access
each machine across the network.

Follow the set up below to configure your machines correctly for this behavior.

> Warning: Changing the default `docker` daemon binding to a TCP port will
> increase your security risks by allowing anyone with access to that port
> full Docker access; so it is not advisable on an open network.
> We strongly encourage enabling [TLS authentication](http://docs.docker.com/swarm/#tls).

## Install Swarm

The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

```bash
$ docker pull swarm
```

## Set up Swarm

1. Log into **each machine** and do the following:

    Start the Docker daemon with the `-H` flag. This ensures that the Docker remote API
    is available over TCP to the *Swarm Manager*.

    ```bash
	$ docker -H tcp://0.0.0.0:2375 -d
    ```

2. Start the Swarm manager on any machine (even on your laptop).

    The simplest way to set up the Swarm manager is to provide a comma-separated
    list of machine addresses:

      ```bash
    	docker run -d -p <manager_port>:2375 swarm manage nodes://<first_machine_address>,<second_machine_addres>,<third_machine_address>,...
      ```

    The machines IP must be accessible from the manager.

    The following example illustrates how to manage a cluster of two machines
    (whose addresses are `172.31.40.100` and `172.31.40.101` respectively) and
    assumes the Docker daemons were started with the `-H tcp://0.0.0.0:2375` flag:

    ```bash
    docker run -d -p <manager_port>:2375 swarm manage nodes://<172.31.40.100:2375>,<172.31.40.101:2375>
    ```

    Replace `<manager_port>` by the TCP port the Swarm manager should listen to.

3. Once the manager is running, check your configuration by running `docker info` as follows:

      ```bash
    	docker -H tcp://<manager_ip:manager_port> info
      ```

      For example, if you run the manager locally on your machine with `<manager_port>` set to `2375`:

      ```bash
      $ docker -H tcp://0.0.0.0:2375 info
      Containers: 0
      Nodes: 2
       machine-1: 172.31.40.101:2375
        └ Containers: 0
        └ Reserved CPUs: 0 / 1
        └ Reserved Memory: 0 B / 514.5 MiB
       machine-0: 172.31.40.100:2375
        └ Containers: 0
        └ Reserved CPUs: 0 / 1
        └ Reserved Memory: 0 B / 514.5 MiB
      ```

    If you are running a test cluster without TLS enabled, you may get an error. In that case, be sure to unset `DOCKER_TLS_VERIFY` with:

      ```bash
    	$ unset DOCKER_TLS_VERIFY
      ```

## Discovery (optional)

Instead of providing a comma-separated list of machines, Swarm can be configured to
dynamically discover machines.

This allows to add and remove machines from the cluster without having to reconfigure and restart
the Swarm manager.

For this to work, each machine will run a Swarm agent. The agent registers the referenced
Docker daemon, monitors it, and updates the discovery backend with the machine's status.

Swarm supports a wide range of discovery backends, including the `Docker Hub`, `consul`, `zookeeper` and `etcd`.

The following example uses the Docker Hub based `token` discovery service:

1. Create a Swarm cluster using the `docker` command.

    ```bash
    $ docker run --rm swarm create
    6856663cdefdec325839a4b7e1de38e8 # <- this is your unique <cluster_id>
    ```

    The `create` command returns a unique cluster ID (`cluster_id`). You'll need
    this ID when starting the Swarm agent on a machine.

2. Log into **each machine** and register the Swarm agents to the discovery service.

    The machine's IP must be accessible from the Swarm Manager.
    Use the following command and replace with the proper `machine_ip` and `cluster_id` to start an agent:

        ```bash
        docker run -d swarm join --addr=<machine_ip:2375> token://<cluster_id>
        ```

        For example:

        ```bash
        $ docker run -d swarm join --addr=172.31.40.100:2375 token://6856663cdefdec325839a4b7e1de38e8
        ```

3. Start the Swarm manager with token discovery:

      ```bash
    	docker run -d -p <swarm_port>:2375 swarm manage token://<cluster_id>
      ```

4. Once the manager is running, check your configuration by running `docker info`.

## Using the docker CLI

You can now use the regular Docker CLI to access your cluster:

```bash
docker -H tcp://<manager_ip:manager_port> info
docker -H tcp://<manager_ip:manager_port> run ...
docker -H tcp://<manager_ip:manager_port> ps
docker -H tcp://<manager_ip:manager_port> logs ...
```

## List machines part of the cluster

When using dynamic discovery, you can get a list of all machines using the `swarm list` command:

```bash
docker run --rm swarm list token://<cluster_id>
<machine_ip:2375>
```

For example:

```bash
$ docker run --rm swarm list token://6856663cdefdec325839a4b7e1de38e8
172.31.40.100:2375
172.31.40.101:2375
172.31.40.102:2375
```

## TLS

Swarm supports TLS authentication between the CLI and Swarm but also between
Swarm and the Docker Engines. _However_, all the Docker daemon certificates and client
certificates **must** be signed using the same CA-certificate.

In order to enable TLS for both client and server, the same command line options
as Docker can be specified:

```bash
swarm manage --tlsverify --tlscacert=<CACERT> --tlscert=<CERT> --tlskey=<KEY> [...]
```

Please refer to the [Docker documentation](https://docs.docker.com/articles/https/)
for more information on how to set up TLS authentication on Docker and generating
the certificates.

> **Note**: Swarm certificates must be generated with `extendedKeyUsage = clientAuth,serverAuth`.

## Discovery services

See the [Discovery service](https://docs.docker.com/swarm/discovery/) document
for more information.

## Advanced Scheduling

See [filters](https://docs.docker.com/swarm/scheduler/filter/) and [strategies](https://docs.docker.com/swarm/scheduler/strategy/) to learn
more about advanced scheduling.

## Swarm API

The [Docker Swarm API](https://docs.docker.com/swarm/API/) is compatible with
the [Docker remote
API](http://docs.docker.com/reference/api/docker_remote_api/), and extends it
with some new endpoints.

## Getting help

Docker Swarm is still in its infancy and under active development. If you need
help, would like to contribute, or simply want to talk about the project with
like-minded individuals, we have a number of open channels for communication.

* To report bugs or file feature requests: please use the [issue tracker on Github](https://github.com/docker/machine/issues).

* To talk about the project with people in real time: please join the `#docker-swarm` channel on IRC.

* To contribute code or documentation changes: please submit a [pull request on Github](https://github.com/docker/machine/pulls).

For more information and resources, please visit the [Getting Help project page](https://docs.docker.com/project/get-help/).
