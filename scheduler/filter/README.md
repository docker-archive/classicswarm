Filters
=======

The `Docker Swarm` scheduler comes with multiple filters.

Thoses filters are used to schedule containers on a subset of nodes.

`Docker Swarm` currently supports 3 filters:
* [Constraint](README.md#constraint-filter)
* [Port](README.md#port-filter)
* [Healty](README.md#healthy-filter)

## Constraint Filter

Constraints are key/value pairs associated to particular nodes. You can see them as *node tags*.

When creating a container, the user can select a subset of nodes that should be considered for scheduling by specifying one or more sets of matching key/value pairs.

This approach has several practical use cases such as:
* Selecting specific host properties (such as `storage=ssd`, in order to schedule containers on specific hardware).
* Tagging nodes based on their physical location (`region=us-east`, to force containers to run on a given location).
* Logical cluster partioning (`environment=production`, to split a cluster into sub-clusters with different properties).

To tag a node with a specific set of key/value pairs, one must pass a list of `--label` options at docker  startup time.

For instance, let's start `node-1` with the `storage=ssd` label:

```bash
$ docker -d --label storage=ssd
$ swarm join --discovery token://XXXXXXXXXXXXXXXXXX --addr=192.168.0.42:2375
```

Again, but this time `node-2` with `storage=disk`:

```bash
$ docker -d --label storage=disk
$ swarm join --discovery token://XXXXXXXXXXXXXXXXXX --addr=192.168.0.43:2375
```

Once the nodes are registered with the cluster, the master pulls their respective tags and will take them into account when scheduling new containers.

Let's start a MySQL server and make sure it gets good I/O performance by selecting nodes with flash drives:

```
$ docker run -d -P -e constraint:storage=ssd --name db mysql
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1      db
```

In this case, the master selected all nodes that met the `storage=ssd` constraint and applied resource management on top of them, as discussed earlier.
`node-1` was selected in this example since it's the only host running flash.

Now we want to run an `nginx` frontend in our cluster. However, we don't want *flash* drives since we'll mostly write logs to disk.

```
$ docker run -d -P -e constraint:storage=disk --name frontend nginx
f8b693db9cd6

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.43:49177->80/tcp      node-2      frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1      db
```

The scheduler selected `node-2` since it was started with the `storage=disk` label.

#### Standard Constraints

Additionally, a standard set of constraints can be used when scheduling containers without specifying them when starting the node.
Those tags are sourced from `docker info` and currently include:

* storagedriver
* executiondriver
* kernelversion
* operatingsystem

## Port Filter

With this filter, `ports` are considered as a unique resource.

```
$ docker run -d -p 80:80 nginx
87c4376856a8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1      prickly_engelbart
```

Docker cluster selects a node where the public `80` port is available and schedules a container on it, in this case `node-1`.

Attempting to run another container with the public `80` port will result in clustering selecting a different node, since that port is already occupied on `node-1`:
```
$ docker run -d -p 80:80 nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.43:80->80/tcp         node-2      dreamy_turing
87c4376856a8        nginx:latest        "nginx"             Up About a minute        running             192.168.0.42:80->80/tcp         node-1      prickly_engelbart
```

Again, repeating the same command will result in the selection of `node-3`, since port `80` is neither available on `node-1` nor `node-2`:
```
$ docker run -d -p 80:80 nginx
963841b138d8

$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NODE        NAMES
f8b693db9cd6        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.44:80->80/tcp         node-3      stoic_albattani
963841b138d8        nginx:latest        "nginx"             Up About a minute        running             192.168.0.43:80->80/tcp         node-2      dreamy_turing
87c4376856a8        nginx:latest        "nginx"             Up About a minute        running             192.168.0.42:80->80/tcp         node-1      prickly_engelbart
```

Finally, Docker Cluster will refuse to run another container that requires port `80` since not a single node in the cluster has it available:
```
$ docker run -d -p 80:80 nginx
2014/10/29 00:33:20 Error response from daemon: no resources availalble to schedule container
```

## Health Filter

This filter will prevent scheduling containers on unhealthy nodes.
