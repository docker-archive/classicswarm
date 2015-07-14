---
page_title: Docker Swarm strategies
page_description: Swarm strategies
page_keywords: docker, swarm, clustering, strategies
---

# Strategies

The Docker Swarm scheduler features multiple strategies for ranking nodes. The
strategy you choose determines how Swarm computes ranking. When you run a new
container, Swarm chooses to place it on the node with the highest computed ranking
for your chosen strategy.

To choose a ranking strategy, pass the `--strategy` flag and a strategy value to
the `swarm manage` command. Swarm currently supports these values:

* `spread`
* `binpack`
* `random`

The `spread` and `binpack` strategies compute rank according to a node's
available CPU, its RAM, and the number of containers it is running. The `random`
strategy uses no computation. It selects a node at random and is primarily
intended for debugging.

Your goal in choosing a strategy is to best optimize your swarm according to
your company's needs.

Under the `spread` strategy, Swarm optimizes for the node with the fewest running containers.
The `binpack` strategy causes Swarm to optimize for the node which is most packed.
The `random` strategy, like it sounds, chooses nodes at random regardless of their available CPU or RAM.

Using the `spread` strategy results in containers spread thinly over many
machines. The advantage of this strategy is that if a node goes down you only
lose a few containers.

The `binpack` strategy avoids fragmentation because it leaves room for bigger
containers on unused machines. The strategic advantage of `binpack` is that you
use fewer machines as Swarm tries to pack as many containers as it can on a
node.

If you do not specify a `--strategy` Swarm uses `spread` by default.

## Spread strategy example

In this example, your swarm is using the `spread` strategy which optimizes for
nodes that have the fewest containers. In this swarm, both `node-1` and `node-2`
have 2G of RAM, 2 CPUs, and neither node is running a container. Under this strategy
`node-1` and `node-2` have the same ranking.

When you run a new container, the system chooses `node-1` at random from the swarm
of two equally ranked nodes:

```bash
$ docker run -d -P -m 1G --name db mysql
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1      db
```

Now, we start another container and ask for 1G of RAM again.

```bash
$ docker run -d -P -m 1G --name frontend nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:49177->80/tcp      node-2      frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1      db
```

The container `frontend` was started on `node-2` because it was the node with the
fewest running containers. If two nodes have the same amount of available RAM and
CPUs, the `spread` strategy prefers the node with the fewest containers running.

## BinPack strategy example

In this example, let's says that both `node-1` and `node-2` have 2G of RAM and
neither is running a container. Again, the nodes are equal. When you run a new
container, the system chooses `node-1` at random from the swarm:

```bash
$ docker run -d -P -m 1G --name db mysql
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1      db
```

Now, you start another container, asking for 1G of RAM again.

```bash
$ docker run -d -P -m 1G --name frontend nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:49177->80/tcp      node-1      frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1      db
```

The system starts the new `frontend` container on `node-1` because it was the
node with the most running containers. This allows us to start a container requiring 2G
of RAM on `node-2`.

If two nodes have the same amount of available RAM and CPUs, the `binpack`
strategy prefers the node with the most running containers.

## Docker Swarm documentation index

- [User guide](https://docs.docker.com/swarm/)
- [Discovery options](https://docs.docker.com/swarm/discovery/)
- [Scheduler filters](https://docs.docker.com/swarm/scheduler/filter/)
- [Swarm API](https://docs.docker.com/swarm/API/)
