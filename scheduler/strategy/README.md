---
page_title: Docker Swarm strategies
page_description: Swarm strategies
page_keywords: docker, swarm, clustering, strategies
---

# Strategies

The `Docker Swarm` scheduler comes with multiple strategies.

These strategies are used to rank nodes using a scores computed by the strategy.

`Docker Swarm` currently supports 2 strategies:
* [BinPacking](#binpacking-strategy)
* [Random](#random-strategy)

You can choose the strategy you want to use with the `--strategy` flag of `swarm manage`

## BinPacking strategy

The BinPacking strategy will rank the nodes using their CPU and RAM available and will return the
node the most packed already. This avoid fragmentation, it will leave room for bigger containers
on unused machines.

For instance, let's says that both `node-1` and `node-2` have 2G of RAM:

```bash
$ docker run -d -P -m 1G --name db mysql
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1      db
```

In this case, `node-1` was chosen randomly, because no container were running, so `node-1` and
`node-2` had the same score.

Now we start another container, asking for 1G of RAM again.

```bash
$ docker run -d -P -m 1G --name frontend nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:49177->80/tcp      node-1      frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1      db
```

The container `frontend` was also started on `node-1` because it was the node the most packed
already. This allows us to start a container requiring 2G of RAM on `node-2`.

## Random strategy

The Random strategy, as it's name says, chooses a random node, it's used mainly for debug.

## Docker Swarm documentation index


- [User guide](./../index.md)
- [Discovery options](./../discovery.md)
- [Scheduler filters](./filter.md)
- [Swarm API](./../API.md)
