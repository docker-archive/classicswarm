---
advisory: swarm-standalone
hide_from_sitemap: true
description: Try Swarm at scale
keywords: docker, swarm, scale, voting, application, certificates
redirect_from:
- /swarm/swarm_at_scale/03-create-cluster/
- /swarm/swarm_at_scale/02-deploy-infra/
title: Deploy application infrastructure
---

In this step, you create several Docker hosts to run your application stack on.
Before you continue, make sure you have taken the time to
[learn the application architecture](about.md).

## About these instructions

This example assumes you are running on a Mac or Windows system and enabling
Docker Engine `docker` commands by provisioning local VirtualBox virtual
machines using Docker Machine. For this evaluation installation, you need 6 (six)
VirtualBox VMs.

While this example uses Docker Machine, this is only one example of an
infrastructure you can use. You can create the environment design on whatever
infrastructure you wish. For example, you could place the application on another
public cloud platform such as Azure or DigitalOcean, on premises in your data
center, or even in a test environment on your laptop.

Finally, these instructions use some common `bash` command substitution techniques to
resolve some values, for example:

```bash
$ eval $(docker-machine env keystore)
```

In a Windows environment, these substitution fail. If you are running in
Windows, replace the substitution `$(docker-machine env keystore)` with the
actual value.


## Task 1. Create the keystore server

To enable a Docker container network and Swarm discovery, you must
deploy (or supply) a key-value store. As a discovery backend, the key-value store
maintains an up-to-date list of cluster members and shares that list with the
Swarm manager. The Swarm manager uses this list to assign tasks to the nodes.

An overlay network requires a key-value store. The key-value store holds
information about the network state which includes discovery, networks,
endpoints, IP addresses, and more.

[Several different backends are supported](../discovery.md). This example uses <a
href="https://www.consul.io/" target="blank">Consul</a> container.

1. Create a "machine" named `keystore`.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=consul"  keystore
   ```

   You can set options for the Engine daemon with the `--engine-opt` flag. In
   this command, you use it to label this Engine instance.

2. Set your local shell to the `keystore` Docker host.

   ```bash
   $ eval $(docker-machine env keystore)
   ```

3. Run [the consul container](https://hub.docker.com/r/progrium/consul/){:target="_blank" class="_"}.

   ```bash
   $ docker run --restart=unless-stopped -d -p 8500:8500 -h consul progrium/consul -server -bootstrap
   ```

   The `-p` flag publishes port 8500 on the container which is where the Consul
   server listens. The server also has several other ports exposed which you can
   see by running `docker ps`.

   ```bash
   $ docker ps
   CONTAINER ID        IMAGE               ...       PORTS                                                                            NAMES
   372ffcbc96ed        progrium/consul     ...       53/tcp, 53/udp, 8300-8302/tcp, 8400/tcp, 8301-8302/udp, 0.0.0.0:8500->8500/tcp   dreamy_ptolemy
   ```

4. Use a `curl` command to test the server by listing the nodes.

   ```bash
   $ curl $(docker-machine ip keystore):8500/v1/catalog/nodes
   [{"Node":"consul","Address":"172.17.0.2"}]
   ```


## Task 2. Create the Swarm manager

In this step, you create the Swarm manager and connect it to the `keystore`
instance. The Swarm manager container is the heart of your Swarm cluster.
It is responsible for receiving all Docker commands sent to the cluster, and for
scheduling resources against the cluster. In a real-world production deployment,
you should configure additional replica Swarm managers as secondaries for high
availability (HA).

Use the `--eng-opt` flag to set the `cluster-store` and
`cluster-advertise` options to refer to the `keystore` server. These options
support the container network you create later.

1. Create the `manager` host.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=manager" \
   --engine-opt="cluster-store=consul://$(docker-machine ip keystore):8500" \
   --engine-opt="cluster-advertise=eth1:2376" manager
   ```

   You also give the daemon a `manager` label.

2. Set your local shell to the `manager` Docker host.

   ```bash
   $ eval $(docker-machine env manager)
   ```

3. Start the Swarm manager process.

   ```bash
   $ docker run --restart=unless-stopped -d -p 3376:2375 \
   -v /var/lib/boot2docker:/certs:ro \
   swarm manage --tlsverify \
   --tlscacert=/certs/ca.pem \
   --tlscert=/certs/server.pem \
   --tlskey=/certs/server-key.pem \
   consul://$(docker-machine ip keystore):8500
   ```

   This command uses the TLS certificates created for the `boot2docker.iso` or
   the manager. This is key for the manager when it connects to other machines
   in the cluster.

4. Test your work by displaying the Docker daemon logs from the host.

   ```bash
   $ docker-machine ssh manager
   <-- output snipped -->
   docker@manager:~$ tail /var/lib/boot2docker/docker.log
   time="2016-04-06T23:11:56.481947896Z" level=debug msg="Calling GET /v1.15/version"
   time="2016-04-06T23:11:56.481984742Z" level=debug msg="GET /v1.15/version"
   time="2016-04-06T23:12:13.070231761Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:12:33.069387215Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:12:53.069471308Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:13:13.069512320Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:13:33.070021418Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:13:53.069395005Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:14:13.071417551Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   time="2016-04-06T23:14:33.069843647Z" level=debug msg="Watch triggered with 1 nodes" discovery=consul
   ```

   The output indicates that the `consul` and the `manager` are communicating correctly.

5. Exit the Docker host.

   ```bash
   docker@manager:~$ exit
   ```

## Task 3. Add the load balancer

The application uses [Interlock](https://github.com/ehazlett/interlock) and Nginx as a
loadbalancer. Before you build the load balancer host, create the
configuration for Nginx.

1. On your local host, create a `config` directory.

2. Change directories to the `config` directory.

   ```bash
   $ cd config
   ```

3. Get the IP address of the Swarm manager host.

   For example:

   ```bash
   $ docker-machine ip manager
   192.168.99.101
   ```

5. Use your favorite editor to create a `config.toml` file and add this content
to the file:

   ```json
   ListenAddr = ":8080"
   DockerURL = "tcp://SWARM_MANAGER_IP:3376"
   TLSCACert = "/var/lib/boot2docker/ca.pem"
   TLSCert = "/var/lib/boot2docker/server.pem"
   TLSKey = "/var/lib/boot2docker/server-key.pem"

   [[Extensions]]
   Name = "nginx"
   ConfigPath = "/etc/nginx/nginx.conf"
   PidPath = "/var/run/nginx.pid"
   MaxConn = 1024
   Port = 80
   ```

6. In the configuration, replace the `SWARM_MANAGER_IP` with the `manager` IP you got
in Step 4.

   You use this value because the load balancer listens on the manager's event
   stream.

7. Save and close the `config.toml` file.

8. Create a machine for the load balancer.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=interlock" loadbalancer
   ```

9. Switch the environment to the `loadbalancer`.

   ```bash
   $ eval $(docker-machine env loadbalancer)
   ```

10. Start an `interlock` container running.

    ```bash
    $ docker run \
        -P \
        -d \
        -ti \
        -v nginx:/etc/conf \
        -v /var/lib/boot2docker:/var/lib/boot2docker:ro \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v $(pwd)/config.toml:/etc/config.toml \
        --name interlock \
        ehazlett/interlock:1.0.1 \
        -D run -c /etc/config.toml
    ```

    This command relies on the `config.toml` file being in the current directory.  After running the command, confirm the image is running:

    ```bash
    $ docker ps
    CONTAINER ID        IMAGE                      COMMAND                  CREATED             STATUS              PORTS                     NAMES
    d846b801a978        ehazlett/interlock:1.0.1   "/bin/interlock -D ru"   2 minutes ago       Up 2 minutes        0.0.0.0:32770->8080/tcp   interlock
    ```

    If you don't see the image running, use `docker ps -a` to list all images to make sure the system attempted to start the image. Then, get the logs to see why the container failed to start.

    ```bash
    $ docker logs interlock
    INFO[0000] interlock 1.0.1 (000291d)
    DEBU[0000] loading config from: /etc/config.toml
    FATA[0000] read /etc/config.toml: is a directory
    ```

    This error usually means you weren't starting the `docker run` from the same
    `config` directory where the `config.toml` file is. If you run the command
    and get a Conflict error such as:

    ```bash
    docker: Error response from daemon: Conflict. The name "/interlock" is already in use by container d846b801a978c76979d46a839bb05c26d2ab949ff9f4f740b06b5e2564bae958. You have to remove (or rename) that container to reuse that name.
    ```

     Remove the interlock container with the `docker container rm interlock` and try again.


11. Start an `nginx` container on the load balancer.

    ```bash
    $ docker run -ti -d \
      -p 80:80 \
      --label interlock.ext.name=nginx \
      --link=interlock:interlock \
      -v nginx:/etc/conf \
      --name nginx \
      nginx nginx -g "daemon off;" -c /etc/conf/nginx.conf
    ```


## Task 4. Create the other Swarm nodes

A host in a Swarm cluster is called a *node*. You've already created the manager
node. Here, the task is to create each virtual host for each node. There are
three commands required:

* create the host with Docker Machine
* point the local environment to the new host
* join the host to the Swarm cluster

If you were building this in a non-Mac/Windows environment, you'd only need to
run the `join` command to add a node to the Swarm cluster and register it with
the Consul discovery service. When you create a node, you also give it a label,
for example:

```
--engine-opt="label=com.function=frontend01"
```

These labels are used later when starting application containers. In the
commands below, notice the label you are applying to each node.


1. Create the `frontend01` host and add it to the Swarm cluster.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=frontend01" \
   --engine-opt="cluster-store=consul://$(docker-machine ip keystore):8500" \
   --engine-opt="cluster-advertise=eth1:2376" frontend01
   $ eval $(docker-machine env frontend01)
   $ docker run -d swarm join --addr=$(docker-machine ip frontend01):2376 consul://$(docker-machine ip keystore):8500
   ```

2. Create the `frontend02` VM.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=frontend02" \
   --engine-opt="cluster-store=consul://$(docker-machine ip keystore):8500" \
   --engine-opt="cluster-advertise=eth1:2376" frontend02
   $ eval $(docker-machine env frontend02)
   $ docker run -d swarm join --addr=$(docker-machine ip frontend02):2376 consul://$(docker-machine ip keystore):8500
   ```

3. Create the `worker01` VM.

   ```bash
   $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
   --engine-opt="label=com.function=worker01" \
   --engine-opt="cluster-store=consul://$(docker-machine ip keystore):8500" \
   --engine-opt="cluster-advertise=eth1:2376" worker01
   $ eval $(docker-machine env worker01)
   $ docker run -d swarm join --addr=$(docker-machine ip worker01):2376 consul://$(docker-machine ip keystore):8500
   ```

4.  Create the `dbstore` VM.

    ```bash
    $ docker-machine create -d virtualbox --virtualbox-memory "2000" \
    --engine-opt="label=com.function=dbstore" \
    --engine-opt="cluster-store=consul://$(docker-machine ip keystore):8500" \
    --engine-opt="cluster-advertise=eth1:2376" dbstore
    $ eval $(docker-machine env dbstore)
    $ docker run -d swarm join --addr=$(docker-machine ip dbstore):2376 consul://$(docker-machine ip keystore):8500
    ```

5.  Check your work.

    At this point, you have deployed on the infrastructure you need to run the
    application. Test this now by listing the running machines:

    ```bash
    $ docker-machine ls
    NAME           ACTIVE   DRIVER       STATE     URL                         SWARM   DOCKER    ERRORS
    dbstore        -        virtualbox   Running   tcp://192.168.99.111:2376           v1.10.3
    frontend01     -        virtualbox   Running   tcp://192.168.99.108:2376           v1.10.3
    frontend02     -        virtualbox   Running   tcp://192.168.99.109:2376           v1.10.3
    keystore       -        virtualbox   Running   tcp://192.168.99.100:2376           v1.10.3
    loadbalancer   -        virtualbox   Running   tcp://192.168.99.107:2376           v1.10.3
    manager        -        virtualbox   Running   tcp://192.168.99.101:2376           v1.10.3
    worker01       *        virtualbox   Running   tcp://192.168.99.110:2376           v1.10.3
    ```

6.  Make sure the Swarm manager sees all your nodes.

    ```
    $ docker -H $(docker-machine ip manager):3376 info
    Containers: 4
     Running: 4
     Paused: 0
     Stopped: 0
    Images: 3
    Server Version: swarm/1.1.3
    Role: primary
    Strategy: spread
    Filters: health, port, dependency, affinity, constraint
    Nodes: 4
     dbstore: 192.168.99.111:2376
      └ Status: Healthy
      └ Containers: 1
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.004 GiB
      └ Labels: com.function=dbstore, executiondriver=native-0.2, kernelversion=4.1.19-boot2docker, operatingsystem=Boot2Docker 1.10.3 (TCL 6.4.1); master : 625117e - Thu Mar 10 22:09:02 UTC 2016, provider=virtualbox, storagedriver=aufs
      └ Error: (none)
      └ UpdatedAt: 2016-04-07T18:25:37Z
     frontend01: 192.168.99.108:2376
      └ Status: Healthy
      └ Containers: 1
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.004 GiB
      └ Labels: com.function=frontend01, executiondriver=native-0.2, kernelversion=4.1.19-boot2docker, operatingsystem=Boot2Docker 1.10.3 (TCL 6.4.1); master : 625117e - Thu Mar 10 22:09:02 UTC 2016, provider=virtualbox, storagedriver=aufs
      └ Error: (none)
      └ UpdatedAt: 2016-04-07T18:26:10Z
     frontend02: 192.168.99.109:2376
      └ Status: Healthy
      └ Containers: 1
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.004 GiB
      └ Labels: com.function=frontend02, executiondriver=native-0.2, kernelversion=4.1.19-boot2docker, operatingsystem=Boot2Docker 1.10.3 (TCL 6.4.1); master : 625117e - Thu Mar 10 22:09:02 UTC 2016, provider=virtualbox, storagedriver=aufs
      └ Error: (none)
      └ UpdatedAt: 2016-04-07T18:25:43Z
     worker01: 192.168.99.110:2376
      └ Status: Healthy
      └ Containers: 1
      └ Reserved CPUs: 0 / 1
      └ Reserved Memory: 0 B / 2.004 GiB
      └ Labels: com.function=worker01, executiondriver=native-0.2, kernelversion=4.1.19-boot2docker, operatingsystem=Boot2Docker 1.10.3 (TCL 6.4.1); master : 625117e - Thu Mar 10 22:09:02 UTC 2016, provider=virtualbox, storagedriver=aufs
      └ Error: (none)
      └ UpdatedAt: 2016-04-07T18:25:56Z
    Plugins:
     Volume:
     Network:
    Kernel Version: 4.1.19-boot2docker
    Operating System: linux
    Architecture: amd64
    CPUs: 4
    Total Memory: 8.017 GiB
    Name: bb13b7cf80e8
    ```

    The command is acting on the Swarm port, so it returns information about the
    entire cluster. You have a manager and no nodes.

## Next step

Your key-value store, load balancer, and Swarm cluster infrastructure are up. You are
ready to [build and run the voting application](deploy-app.md) on it.
