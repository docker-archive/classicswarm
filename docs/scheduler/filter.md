<!--[metadata]>
+++
title = "Filters"
description = "Swarm filters"
keywords = ["docker, swarm, clustering,  filters"]
[menu.main]
parent="swarm_sched"
weight=4
+++
<![end-metadata]-->

# Swarm filters

Filters tell Docker Swarm scheduler which nodes to use when creating and running
a container.

## Configure the available filters

Filters are divided into two categories, node filters and container configuration
filters. Node filters operate on characteristics of the Docker host or on the
configuration of the Docker daemon. Container configuration filters operate on
characteristics of containers, or on the availability of images on a host.

Each filter has a name that identifies it. The node filters are:

* `constraint`
* `health`

The container configuration filters are:

* `affinity`
* `dependency`
* `port`

When you start a Swarm manager with the `swarm manage` command, all the filters
are enabled. If you want to limit the filters available to your Swarm, specify a subset
of filters by passing the `--filter` flag and the name:

```bash
$ swarm manage --filter=health --filter=dependency
```

> **Note**: Container configuration filters match all containers, including stopped
> containers, when applying the filter. To release a node used by a container, you
> must remove the container from the node.

## Node filters

When creating a container or building an image, you use a `constraint` or
`health` filter to select a subset of nodes to consider for scheduling.

### Use a constraint filter

Node constraints can refer to Docker's default tags or to custom labels. Default
tags are sourced from `docker info`. Often, they relate to properties of the Docker
host. Currently, the default tags include:

* `node` to refer to the node by ID or name
* `storagedriver`
* `executiondriver`
* `kernelversion`
* `operatingsystem`

Custom node labels you apply when you start the `docker daemon`, for example:

```bash
$ docker daemon --label com.example.environment="production" --label
com.example.storage="ssd"
```

Then, when you start a container on the cluster, you can set constraints using
these default tags or custom labels. The Swarm scheduler looks for matching node
on the cluster and starts the container there. This approach has several
practical applications:

* Schedule based on specific host properties, for example,`storage=ssd` schedules
  containers on specific hardware.
* Force containers to run in a given location, for example region=us-east`.
* Create logical cluster partitions by splitting a cluster into
  sub-clusters with different properties, for example `environment=production`.


#### Example node constraints  

To specify custom label for a node, pass a list of `--label`
options at `docker` startup time. For instance, to start `node-1` with the
`storage=ssd` label:

```bash
$ docker daemon --label storage=ssd
$ swarm join --advertise=192.168.0.42:2375 token://XXXXXXXXXXXXXXXXXX
```

You might start a different `node-2` with `storage=disk`:

```bash
$ docker daemon --label storage=disk
$ swarm join --advertise=192.168.0.43:2375 token://XXXXXXXXXXXXXXXXXX
```

Once the nodes are joined to a cluster, the Swarm manager pulls their respective
tags.  Moving forward, the manager takes the tags into account when scheduling
new containers.

Continuing the previous example, assuming your cluster with `node-1` and
`node-2`, you can run a MySQL server container on the cluster.  When you run the
container, you can use a `constraint` to ensure the database gets good I/O
performance. You do this by filtering for nodes with flash drives:

```bash
$ docker tcp://<manager_ip:manager_port>  run -d -P -e constraint:storage==ssd --name db mysql
f8b693db9cd6

$ docker tcp://<manager_ip:manager_port>  ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
f8b693db9cd6        mysql:latest        "mysqld"            Less than a second ago   running             192.168.0.42:49178->3306/tcp    node-1/db
```

In this example, the manager selected all nodes that met the `storage=ssd`
constraint and applied resource management on top of them.   Only `node-1` was
selected because it's the only host running flash.

Suppose you want to run an Nginx frontend in a cluster. In this case, you wouldn't want flash drives because the frontend mostly writes logs to disk.

```bash
$ docker tcp://<manager_ip:manager_port> run -d -P -e constraint:storage==disk --name frontend nginx
963841b138d8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
963841b138d8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.43:49177->80/tcp      node-2/frontend
f8b693db9cd6        mysql:latest        "mysqld"            Up About a minute        running             192.168.0.42:49178->3306/tcp    node-1/db
```

The scheduler selected `node-2` since it was started with the `storage=disk` label.

Finally, build args can be used to apply node constraints to a `docker build`.
Again, you'll avoid flash drives.

```bash
$ mkdir sinatra
$ cd sinatra
$ echo "FROM ubuntu:14.04" > Dockerfile
$ echo "MAINTAINER Kate Smith <ksmith@example.com>" >> Dockerfile
$ echo "RUN apt-get update && apt-get install -y ruby ruby-dev" >> Dockerfile
$ echo "RUN gem install sinatra" >> Dockerfile
$ docker build --build-arg=constraint:storage==disk -t ouruser/sinatra:v2 .
Sending build context to Docker daemon 2.048 kB
Step 1 : FROM ubuntu:14.04
 ---> a5a467fddcb8
Step 2 : MAINTAINER Kate Smith <ksmith@example.com>
 ---> Running in 49e97019dcb8
 ---> de8670dcf80e
Removing intermediate container 49e97019dcb8
Step 3 : RUN apt-get update && apt-get install -y ruby ruby-dev
 ---> Running in 26c9fbc55aeb
 ---> 30681ef95fff
Removing intermediate container 26c9fbc55aeb
Step 4 : RUN gem install sinatra
 ---> Running in 68671d4a17b0
 ---> cd70495a1514
Removing intermediate container 68671d4a17b0
Successfully built cd70495a1514

$ docker images
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
dockerswarm/swarm   manager             8c2c56438951        2 days ago          795.7 MB
ouruser/sinatra     v2                  cd70495a1514        35 seconds ago      318.7 MB
ubuntu              14.04               a5a467fddcb8        11 days ago         187.9 MB
```

### Use the health filter

The node `health` filter prevents the scheduler form running containers
on unhealthy nodes. A node is considered unhealthy if the node is down or it
can't communicate with the cluster store.

## Container filters

When creating a container, you can use three types of container filters:

* [`affinity`](#use-an-affinity-filter)
* [`dependency`](#use-a-depedency-filter)
* [`port`](#use-a-port-filter)

### Use an affinity filter

Use an `affinity` filter to create "attractions" between containers. For
example, you can run a container and instruct Swarm to schedule it next to
another container based on these affinities:

* container name or id
* an image on the host
* a custom label applied to the container

These affinities ensure that containers run on the same network node
&mdash; without you having to know what each node is running.

#### Example name affinity

You can schedule a new container to run next to another based on a container
name or ID. For example, you can start a container called `frontend` running
`nginx`:

```bash
$ docker tcp://<manager_ip:manager_port>  run -d -p 80:80 --name frontend nginx
 87c4376856a8


$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1/frontend
```

Then, using `-e affinity:container==frontend` value to schedule a second
container to locate and run next to the container named `frontend`.

```bash
$ docker tcp://<manager_ip:manager_port> run -d --name logger -e affinity:container==frontend logger
 87c4376856a8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1/frontend
963841b138d8        logger:latest       "logger"            Less than a second ago   running                                             node-1/logger
```

Because of `name` affinity, the  `logger` container ends up on `node-1` along
with the `frontend` container. Instead of the `frontend` name you could have
supplied its ID as follows:

```bash
$ docker tcp://<manager_ip:manager_port> run -d --name logger -e affinity:container==87c4376856a8
```

#### Example image affinity

You can schedule a container to run only on nodes where a specific image is
already pulled. For example, suppose you pull a `redis` image to two hosts and a
`mysql` image to a third.

```bash
$ docker -H node-1:2375 pull redis
$ docker -H node-2:2375 pull mysql
$ docker -H node-3:2375 pull redis
```

Only `node-1` and `node-3` have the `redis` image. Specify a `-e
affinity:image==redis` filter to schedule several additional containers to run
on these nodes.

```bash
$ docker tcp://<manager_ip:manager_port> run -d --name redis1 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis2 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis3 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis4 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis5 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis6 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis7 -e affinity:image==redis redis
$ docker tcp://<manager_ip:manager_port> run -d --name redis8 -e affinity:image==redis redis

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
87c4376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1/redis1
1212386856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1/redis2
87c4376639a8        redis:latest        "redis"             Less than a second ago   running                                             node-3/redis3
1234376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-1/redis4
86c2136253a8        redis:latest        "redis"             Less than a second ago   running                                             node-3/redis5
87c3236856a8        redis:latest        "redis"             Less than a second ago   running                                             node-3/redis6
87c4376856a8        redis:latest        "redis"             Less than a second ago   running                                             node-3/redis7
963841b138d8        redis:latest        "redis"             Less than a second ago   running                                             node-1/redis8
```

As you can see here, the containers were only scheduled on nodes that had the
`redis` image. Instead of the image name, you could have specified the image ID.

```bash
$ docker images
REPOSITORY                         TAG                       IMAGE ID            CREATED             VIRTUAL SIZE
redis                              latest                    06a1f75304ba        2 days ago          111.1 MB

$ docker tcp://<manager_ip:manager_port> run -d --name redis1 -e affinity:image==06a1f75304ba redis
```

#### Example label affinity

A label affinity allows you to filter based on a custom container label. For
example, you can run a `nginx` container and apply the
`com.example.type=frontend` custom label.

```bash
$ docker tcp://<manager_ip:manager_port> run -d -p 80:80 --label com.example.type=frontend nginx
 87c4376856a8

$ docker tcp://<manager_ip:manager_port> ps  --filter "label=com.example.type=frontend"
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1/trusting_yonath
```

Then, use `-e affinity:com.example.type==frontend` to schedule a container next
to the container with the `com.example.type==frontend` label.

```bash
$ docker tcp://<manager_ip:manager_port> run -d -e affinity:com.example.type==frontend logger
 87c4376856a8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND             CREATED                  STATUS              PORTS                           NAMES
87c4376856a8        nginx:latest        "nginx"             Less than a second ago   running             192.168.0.42:80->80/tcp         node-1/trusting_yonath
963841b138d8        logger:latest       "logger"            Less than a second ago   running                                             node-1/happy_hawking
```

The `logger` container ends up on `node-1` because its affinity with the
`com.example.type==frontend` label.

### Use a dependency filter

A container dependency filter co-schedules dependent containers on the same node.
Currently, dependencies are declared as follows:

* `--volumes-from=dependency` (shared volumes)
* `--link=dependency:alias` (links)
* `--net=container:dependency` (shared network stacks)

Swarm attempts to co-locate the dependent container on the same node. If it
cannot be done (because the dependent container doesn't exist, or because the
node doesn't have enough resources), it will prevent the container creation.

The combination of multiple dependencies are honored if possible. For
instance, if you specify `--volumes-from=A --net=container:B`,  the scheduler
attempts to co-locate the container on the same node as `A` and `B`. If those
containers are running on different nodes, Swarm does not schedule the container.

### Use a port filter

When the `port` filter is enabled, a container's port configuration is used as a
unique constraint. Docker Swarm selects a node where a particular port is
available and unoccupied by another container or process. Required ports may be
specified by mapping a host port, or using the host networking and exposing a
port using the container configuration.

#### Example in bridge mode

By default, containers run on Docker's bridge network. To use the `port` filter
with the bridge network, you run a container as follows.

```bash
$ docker tcp://<manager_ip:manager_port> run -d -p 80:80 nginx
87c4376856a8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID    IMAGE               COMMAND         PORTS                       NAMES
87c4376856a8    nginx:latest        "nginx"         192.168.0.42:80->80/tcp     node-1/prickly_engelbart
```

Docker Swarm selects a node where port `80` is available and unoccupied by another
container or process, in this case `node-1`. Attempting to run another container
that uses the host port `80` results in Swarm selecting a different node,
because port `80` is already occupied on `node-1`:

```bash
$ docker tcp://<manager_ip:manager_port> run -d -p 80:80 nginx
963841b138d8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE          COMMAND        PORTS                           NAMES
963841b138d8        nginx:latest   "nginx"        192.168.0.43:80->80/tcp         node-2/dreamy_turing
87c4376856a8        nginx:latest   "nginx"        192.168.0.42:80->80/tcp         node-1/prickly_engelbart
```

Again, repeating the same command will result in the selection of `node-3`,
since port `80` is neither available on `node-1` nor `node-2`:

```bash
$ docker tcp://<manager_ip:manager_port> run -d -p 80:80 nginx
963841b138d8

$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID   IMAGE               COMMAND        PORTS                           NAMES
f8b693db9cd6   nginx:latest        "nginx"        192.168.0.44:80->80/tcp         node-3/stoic_albattani
963841b138d8   nginx:latest        "nginx"        192.168.0.43:80->80/tcp         node-2/dreamy_turing
87c4376856a8   nginx:latest        "nginx"        192.168.0.42:80->80/tcp         node-1/prickly_engelbart
```

Finally, Docker Swarm will refuse to run another container that requires port
`80`, because it is not available on any node in the cluster:

```bash
$ docker tcp://<manager_ip:manager_port> run -d -p 80:80 nginx
2014/10/29 00:33:20 Error response from daemon: no resources available to schedule container
```

Each container occupies port `80` on its residing node when the container
is created and releases the port when the container is deleted. A container in `exited`
state still owns the port. If `prickly_engelbart` on `node-1` is stopped but not
deleted, trying to start another container on `node-1` that requires port `80` would fail
because port `80` is associated with `prickly_engelbart`. To increase running
instances of nginx, you can either restart `prickly_engelbart`, or start another container
after deleting `prickly_englbart`.

#### Node port filter with host networking

A container running with `--net=host` differs from the default
`bridge` mode as the `host` mode does not perform any port binding. Instead,
host mode requires that you  explicitly expose one or more port numbers.  You
expose a port using `EXPOSE` in the `Dockerfile` or `--expose` on the command
line. Swarm makes use of this information in conjunction with the `host` mode to
choose an available node for a new container.

For example, the following commands start `nginx` on 3-node cluster.

```bash
$ docker tcp://<manager_ip:manager_port> run -d --expose=80 --net=host nginx
640297cb29a7
$ docker tcp://<manager_ip:manager_port> run -d --expose=80 --net=host nginx
7ecf562b1b3f
$ docker tcp://<manager_ip:manager_port> run -d --expose=80 --net=host nginx
09a92f582bc2
```

Port binding information is not available through the `docker ps` command because
all the nodes were started with the `host` network.

```bash
$ docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND                CREATED                  STATUS              PORTS               NAMES
640297cb29a7        nginx:1             "nginx -g 'daemon of   Less than a second ago   Up 30 seconds                           box3/furious_heisenberg
7ecf562b1b3f        nginx:1             "nginx -g 'daemon of   Less than a second ago   Up 28 seconds                           box2/ecstatic_meitner
09a92f582bc2        nginx:1             "nginx -g 'daemon of   46 seconds ago           Up 27 seconds                           box1/mad_goldstine
```

Swarm refuses the operation when trying to instantiate the 4th container.

```bash
$  docker tcp://<manager_ip:manager_port> run -d --expose=80 --net=host nginx
FATA[0000] Error response from daemon: unable to find a node with port 80/tcp available in the Host mode
```

However, port binding to the different value, for example  `81`, is still allowed.

```bash
$  docker tcp://<manager_ip:manager_port> run -d -p 81:80 nginx:latest
832f42819adc
$  docker tcp://<manager_ip:manager_port> ps
CONTAINER ID        IMAGE               COMMAND                CREATED                  STATUS                  PORTS                                 NAMES
832f42819adc        nginx:1             "nginx -g 'daemon of   Less than a second ago   Up Less than a second   443/tcp, 192.168.136.136:81->80/tcp   box3/thirsty_hawking
640297cb29a7        nginx:1             "nginx -g 'daemon of   8 seconds ago            Up About a minute                                             box3/furious_heisenberg
7ecf562b1b3f        nginx:1             "nginx -g 'daemon of   13 seconds ago           Up About a minute                                             box2/ecstatic_meitner
09a92f582bc2        nginx:1             "nginx -g 'daemon of   About a minute ago       Up About a minute                                             box1/mad_goldstine
```

## How to write filter expressions

To apply a node `constraint` or container `affinity` filters you must set
environment variables on the container using filter expressions, for example:

```bash
$ docker tcp://<manager_ip:manager_port> run -d --name redis1 -e affinity:image==~redis redis
```

Each expression must be in the form:

```
<filter-type>:<key><operator><value>
```

The `<filter-type>` is either the `affinity` or the `constraint` keyword. It
identifies the type filter you intend to use.

The `<key>` is an alpha-numeric and must start with a letter or underscore. The
`<key>` corresponds to one of the following:

* the `container` keyword
* the `node` keyword
* a default tag (node constraints)
* a custom metadata label (nodes or containers).

The `<operator> `is either `==` or `!=`. By default, expression operators are
hard enforced. If an expression is not met exactly , the manager does not
schedule the container. You can use a `~`(tilde) to create a "soft" expression.
The scheduler tries to match a soft expression. If the expression is not met,
the scheduler discards the filter and schedules the container according to the
scheduler's strategy.

The `<value>` is an alpha-numeric string, dots, hyphens, and underscores making
up one of the following:

* A globbing pattern, for example, `abc*`.
* A regular expression in the form of `/regexp/`. See
  [re2 syntax](https://github.com/google/re2/wiki/Syntax) for the supported
  regex syntax.

The following examples illustrate some possible expressions:

* `constraint:node==node1` matches node `node1`.
* `constraint:node!=node1` matches all nodes, except `node1`.
* `constraint:region!=us*` matches all nodes outside with a `region` tag prefixed with `us`.
* `constraint:node==/node[12]/` matches nodes `node1` and `node2`.
* `constraint:node==/node\d/` matches all nodes with `node` + 1 digit.
* `constraint:node!=/node-[01]/` matches all nodes, except `node-0` and `node-1`.
* `constraint:node!=/foo\[bar\]/` matches all nodes, except `foo[bar]`. You can see the use of escape characters here.
* `constraint:node==/(?i)node1/` matches node `node1` case-insensitive. So `NoDe1` or `NODE1` also match.
* `affinity:image==~redis` tries to match for nodes running container with a `redis` image
* `constraint:region==~us*` searches for nodes in the cluster belonging to the `us` region
* `affinity:container!=~redis*` schedule a new `redis5` container to a node
without a container that satisfies `redis*`

## Related information

- [Docker Swarm overview](../index.md)
- [Discovery options](../discovery.md)
- [Scheduler strategies](strategy.md)
- [Swarm API](../swarm-api.md)
