<!--[metadata]>
+++
title = "Create a swarm for development"
description = "Swarm: a Docker-native clustering system"
keywords = ["docker, swarm,  clustering"]
[menu.main]
parent="smn_workw_swarm"
weight=2
+++
<![end-metadata]-->

# Create a swarm for development

This section tells you how to create a Docker Swarm on your network to use only for debugging, testing, or development purposes. You can also use this type of installation if you are developing custom applications for Docker Swarm or contributing to it.  

> **Caution**: Only use this set up if your network environment is secured by a firewall or other measures.

## Prerequisites 

You install Docker Swarm on a single system which is known as your Docker Swarm
manager. You create the cluster, or swarm, on one or more additional nodes on
your network.  Each node in your swarm must:

* be accessible by the swarm manager across your network
* have Docker Engine 1.6.0+ installed
* open a TCP port to listen for the manager
* *do not* install on a VM or from an image created through cloning

Docker generates a unique ID for the Engine that is located in the
`/etc/docker/key.json` file. If a VM is cloned from an instance where a
Docker daemon was previously pre-installed, Swarm will be unable to differentiate
among the remote Docker engines. This is because the cloning process copied the
the identical ID to each image and the ID is no longer unique.

If you forget this restriction and create a node anyway, Swarm displays a single
Docker engine as registered. To workaround this problem, you can generate a new
ID for each node with affected by this issue. To do this stop the daemon on a node,
delete its `/etc/docker/key.json` file, and restart the daemon.

You can run Docker Swarm on Linux 64-bit architectures. You can also install and
run it on 64-bit Windows and Max OSX but these architectures are *not* regularly
tested for compatibility.

Take a moment and identify the systems on your network that you intend to use.
Ensure each node meets the requirements listed above.

## Pull the swarm image and create a cluster.

The easiest way to get started with Swarm is to use the
[official Docker image](https://registry.hub.docker.com/_/swarm/).

1. Pull the swarm image.

		$ docker pull swarm

1. Create a Swarm cluster using the `docker` command.

		$ docker run --rm swarm create
		6856663cdefdec325839a4b7e1de38e8 # 

	The `create` command returns a unique cluster ID (`cluster_id`). You'll need
	this ID when starting the Docker Swarm agent on a node.

##  Create swarm nodes

Each Swarm node will run a Swarm node agent. The agent registers the referenced
Docker daemon, monitors it, and updates the discovery backend with the node's status.

This example uses the Docker Hub based `token` discovery service (only for testing/dev, not for production).
Log into **each node** and do the following.

1. Start the Docker daemon with the `-H` flag. This ensures that the
Docker remote API on *Swarm Agents* is available over TCP for the
*Swarm Manager*, as well as the standard unix socket which is
available in default docker installs.

		$ docker daemon -H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock

	> **Note**: versions of docker prior to 1.8 used the `-d` flag instead of the `docker daemon` subcommand.

2. Register the Swarm agents to the discovery service. The node's IP must be accessible from the Swarm Manager. Use the following command and replace with the proper `node_ip` and `cluster_id` to start an agent:

		docker run -d swarm join --addr=<node_ip:2375> token://<cluster_id>

	For example:

		$ docker run -d swarm join --addr=172.31.40.100:2375 token://6856663cdefdec325839a4b7e1de38e8

## Configure a manager

Once you have your nodes established, set up a manager to control the swarm.

1. Start the Swarm manager on any machine or your laptop. 

	The following command illustrates how to do this:

		docker run -d -p <manager_port>:2375 swarm manage token://<cluster_id>

	The manager is exposed and listening on `<manager_port>`.

2. Once the manager is running, check your configuration by running `docker info` as follows:

		docker -H tcp://<manager_ip:manager_port> info

	For example, if you run the manager locally on your machine:

		$ docker -H tcp://0.0.0.0:2375 info
		Containers: 0
		Nodes: 3
		 agent-2: 172.31.40.102:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB
		 agent-1: 172.31.40.101:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB
		 agent-0: 172.31.40.100:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB

  If you are running a test cluster without TLS enabled, you may get an error.
  In that case, be sure to unset `DOCKER_TLS_VERIFY` with:

    $ unset DOCKER_TLS_VERIFY
  
## Using the docker CLI

You can now use the regular Docker CLI to access your nodes:

	docker -H tcp://<manager_ip:manager_port> info
	docker -H tcp://<manager_ip:manager_port> run ...
	docker -H tcp://<manager_ip:manager_port> ps
	docker -H tcp://<manager_ip:manager_port> logs ...


## List nodes in your cluster

You can get a list of all your running nodes using the `swarm list` command:

	docker run --rm swarm list token://<cluster_id>
	<node_ip:2375>


For example:

	$ docker run --rm swarm list token://6856663cdefdec325839a4b7e1de38e8
	172.31.40.100:2375
	172.31.40.101:2375
	172.31.40.102:2375

## TLS

Swarm supports TLS authentication between the CLI and Swarm but also between
Swarm and the Docker nodes. _However_, all the Docker daemon certificates and client
certificates **must** be signed using the same CA-certificate.

In order to enable TLS for both client and server, the same command line options
as Docker can be specified:


	swarm manage --tlsverify --tlscacert=<CACERT> --tlscert=<CERT> --tlskey=<KEY> [...]


Please refer to the [Docker documentation](https://docs.docker.com/articles/https/)
for more information on how to set up TLS authentication on Docker and generating
the certificates.

> **Note**: Swarm certificates must be generated with `extendedKeyUsage = clientAuth,serverAuth`.
