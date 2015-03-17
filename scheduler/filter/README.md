---
page_title: Docker Swarm filters
page_description: Swarm filters
page_keywords: docker, swarm, clustering, filters
---

# Filters

The `Docker Swarm` scheduler comes with multiple filters.

These filters are used to schedule containers on a subset of nodes.

`Docker Swarm` currently supports 4 filters:
* [Constraint](#constraint-filter)
* [Affinity](#affinity-filter)
* [Port](#port-filter)
* [Health](#health-filter)

You can choose the filter(s) you want to use with the `--filter` flag of `swarm manage`

## Constraint Filter

Constraints are key/value pairs associated to particular nodes. You can see them
as *node tags*.

When creating a container, the user can select a subset of nodes that should be
considered for scheduling by specifying one or more sets of matching key/value pairs.

This approach has several practical use cases such as:
* Selecting specific host properties (such as `storage=ssd`, in order to schedule
  containers on specific hardware).
* Tagging nodes based on their physical location (`region=us-east`, to force
  containers to run on a given location).
* Logical cluster partitioning (`environment=production`, to split a cluster into
  sub-clusters with different properties).

To tag a node with a specific set of key/value pairs, one must pass a list of
`--label` options at docker startup time.

For instance, let's start `node-1` with the `storage=ssd` label:

```bash
$ docker -d --label storage=ssd
$ swarm join --addr=192.168.0.42:2375 token://XXXXXXXXXXXXXXXXXX
```

Again, but this time `node-2` with `storage=disk`:

```bash
$ docker -d --label storage=disk
$ swarm join --addr=192.168.0.43:2375 token://XXXXXXXXXXXXXXXXXX
```

Once the nodes are registered with the cluster, the master pulls their respective
tags and will take them into account when scheduling new containers.

Let's start a MySQL server and make sure it gets good I/O performance by selecting
nodes with flash drives:

```
$ docker run -d -P -e constraint:storage==ssd --name db mysql
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1      db
```

In this case, the master selected all nodes that met the `storage=ssd` constraint
and applied resource management on top of them, as discussed earlier.
`node-1` was selected in this example since it's the only host running flash.

Now we want to run an `nginx` frontend in our cluster. However, we don't want
*flash* drives since we'll mostly write logs to disk.

```
$ docker run -d -P -e constraint:storage==disk --name frontend nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.43:49177->80/tcp      node-2      frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1      db
```

The scheduler selected `node-2` since it was started with the `storage=disk` label.

## Standard Constraints

Additionally, a standard set of constraints can be used when scheduling containers
without specifying them when starting the node. Those tags are sourced from
`docker info` and currently include:

* storagedriver
* executiondriver
* kernelversion
* operatingsystem

## Affinity Filter

#### Containers

You can schedule 2 containers and make the container #2 next to the container #1.

```
$ docker run -d -p 80:80 --name front nginx
 87c4376856a8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1      front
```

Using `-e affinity:container==front` will schedule a container next to the container `front`.
You can also use IDs instead of name: `-e affinity:container==87c4376856a8`

```
$ docker run -d --name logger -e affinity:container==front logger
 87c4376856a8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1      front
963841b138d8        logger:latest       "logger"            Less than a second ago   running                                             node-1      logger
```

The `logger` container ends up on `node-1` because its affinity with the container `front`.

#### Images

You can schedule a container only on nodes where the images are already pulled.

```
$ docker -H node-1:2375 pull redis
$ docker -H node-2:2375 pull mysql
$ docker -H node-3:2375 pull redis
```

Here only `node-1` and `node-3` have the `redis` image. Using `-e affinity:image=redis` we can
schedule container only on these 2 nodes. You can also use the image ID instead of its name.

```
$ docker run -d --name redis1 -e affinity:image==redis redis
$ docker run -d --name redis2 -e affinity:image==redis redis
$ docker run -d --name redis3 -e affinity:image==redis redis
$ docker run -d --name redis4 -e affinity:image==redis redis
$ docker run -d --name redis5 -e affinity:image==redis redis
$ docker run -d --name redis6 -e affinity:image==redis redis
$ docker run -d --name redis7 -e affinity:image==redis redis
$ docker run -d --name redis8 -e affinity:image==redis redis

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
87c4376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1      redis1
1212386856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1      redis2
87c4376639a8        redis:latest        "redis"             Less than a second ago   running                                             node-3      redis3
1234376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1      redis4
86c2136253a8        redis:latest        "redis"             Less than a second ago   running                                             node-3      redis5
87c3236856a8        redis:latest        "redis"             Less than a second ago   running                                             node-3      redis6
87c4376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-3      redis7
963841b138d8        redis:latest        "redis"             Less than a second ago   running                                             node-1      redis8
```

As you can see here, the containers were only scheduled on nodes with the redis image already pulled.

#### Expression Syntax

An affinity or a constraint expression consists of a `key` and a `value`.
A `key` must conform the alpha-numeric pattern, with the leading alphabet or underscore.

A `value` must be one of the following:
* An alpha-numeric string, dots, hyphens, and underscores.
* A globbing pattern, i.e., `abc*`.
* A regular expression in the form of `/regexp/`. We support the Go's regular expression syntax.

Current `swarm` supports affinity/constraint operators as the following: `==` and `!=`.

For example,
* `constraint:node==node1` will match node `node1`.
* `constraint:node!=node1` will match all nodes, except `node1`.
* `constraint:region!=us*` will match all nodes outside the regions prefixed with `us`.
* `constraint:node==/node[12]/` will match nodes `node1` and `node2`.
* `constraint:node==/node\d/` will match all nodes with `node` + 1 digit.
* `constraint:node!=/node-[01]/` will match all nodes, except `node-0` and `node-1`.
* `constraint:node!=/foo\[bar\]/` will match all nodes, except `foo[bar]`. You can see the use of escape characters here.
* `constraint:node==/(?i)node1/` will match node `node1` case-insensitive. So 'NoDe1' or 'NODE1' will also match.

#### Soft Affinities

By default, affinities are hard enforced. If an affinity is not met, the container won't be scheduled.
With soft affinites the scheduler will try to meet the affinity. If it is not met, the scheduler will discard the filter and schedule the container according to the scheduler strategy.

soft affinites are expressed with a **~** in the expression
Example ,
```
$ docker run -d --name redis1 -e affinity:image==~redis redis
```
If none of the nodes in the cluster has image redis, the scheduler will discard the affinity and schedules according to the strategy.

```
$ docker run -d --name redis5 -e affinity:container!=~redis* redis
```
The affinity filter will be used to schedule a new redis5 container to a different node that doesn't have a container with the name that satisfies redis*. If each node in the cluster has a redis* container, the scheduler will discard the affinity rule and schedules according to the strategy. 

## Port Filter

With this filter, `ports` are considered as unique resources.

```
$ docker run -d -p 80:80 nginx
87c4376856a8

$ docker ps
CONTAINER ID    IMAGE               COMMAND         PORTS                       NODE        NAMES
87c4376856a8    nginx:latest        "nginx"         192.168.0.42:80->80/tcp     node-1      prickly_engelbart
```

Docker cluster selects a node where the public `80` port is available and schedules
a container on it, in this case `node-1`.

Attempting to run another container with the public `80` port will result in
clustering selecting a different node, since that port is already occupied on `node-1`:

```
$ docker run -d -p 80:80 nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE          COMMAND        PORTS                           NODE        NAMES
963841b138d8        nginx:latest   "nginx"        192.168.0.43:80->80/tcp         node-2      dreamy_turing
87c4376856a8        nginx:latest   "nginx"        192.168.0.42:80->80/tcp         node-1      prickly_engelbart
```

Again, repeating the same command will result in the selection of `node-3`, since
port `80` is neither available on `node-1` nor `node-2`:

```
$ docker run -d -p 80:80 nginx
963841b138d8

$ docker ps
CONTAINER ID   IMAGE               COMMAND        PORTS                           NODE        NAMES
f8b693db9cd6   nginx:latest        "nginx"        192.168.0.44:80->80/tcp         node-3      stoic_albattani
963841b138d8   nginx:latest        "nginx"        192.168.0.43:80->80/tcp         node-2      dreamy_turing
87c4376856a8   nginx:latest        "nginx"        192.168.0.42:80->80/tcp         node-1      prickly_engelbart
```

Finally, Docker Cluster will refuse to run another container that requires port
`80` since not a single node in the cluster has it available:

```
$ docker run -d -p 80:80 nginx
2014/10/29 00:33:20 Error response from daemon: no resources available to schedule container
```

## Dependency Filter

This filter co-schedules dependent containers on the same node.

Currently, dependencies are declared as follows:

- Shared volumes: `--volumes-from=dependency`
- Links: `--link=dependency:alias`
- Shared network stack: `--net=container:dependency`

Swarm will attempt to co-locate the dependent container on the same node. If it
cannot be done (because the dependent container doesn't exist, or because the
node doesn't have enough resources), it will prevent the container creation.

The combination of multiple dependencies will be honored if possible. For
instance, `--volumes-from=A --net=container:B` will attempt to co-locate the
container on the same node as `A` and `B`. If those containers are running on
different nodes, Swarm will prevent you from scheduling the container.

## Health Filter

This filter will prevent scheduling containers on unhealthy nodes.

## Docker Swarm documentation index

- [User guide](./../index.md)
- [Discovery options](./../discovery.md)
- [Sheduler strategies](./strategy.md)
- [Swarm API](./../API.md)
