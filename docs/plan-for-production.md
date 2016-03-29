<!--[metadata]>
+++
title = "Plan for Swarm in production"
description = "Plan for Swarm in production"
keywords = ["docker, swarm, scale, voting, application,  plan"]
[menu.main]
parent="workw_swarm"
weight=-45
+++
<![end-metadata]-->

# Plan for Swarm in production

This article provides guidance to help you plan, deploy, and manage Docker
Swarm clusters in business critical production environments. The following high
 level topics are covered:

- [Security](#security)
- [High Availability](#high-availability)
- [Performance](#performance)
- [Cluster ownership](#cluster-ownership)

## Security

There are many aspects to securing a Docker Swarm cluster. This section covers:

- Authentication using TLS
- Network access control

These topics are not exhaustive. They form part of a wider security architecture
that includes: security patching, strong password policies, role based access
control, technologies such as SELinux and AppArmor, strict auditing, and more.

### Configure Swarm for TLS

All nodes in a Swarm cluster must bind their Docker Engine daemons to a network
port. This brings with it all of the usual network related security
implications such as man-in-the-middle attacks. These risks are compounded when
 the network in question is untrusted such as the internet. To mitigate these
risks, Swarm and the Engine support Transport Layer Security(TLS) for
authentication.

The Engine daemons, including the Swarm manager, that are configured to use TLS
will only accept commands from Docker Engine clients that sign their
communications. The Engine and Swarm support external 3rd party Certificate
Authorities (CA) as well as internal corporate CAs.

The default Engine and Swarm ports for TLS are:

- Engine daemon: 2376/tcp
- Swarm manager: 3376/tcp

For more information on configuring Swarm for TLS, see the [Overview Docker Swarm with TLS](secure-swarm-tls.md) page.

### Network access control

Production networks are complex, and usually locked down so that only allowed
traffic can flow on the network. The list below shows the network ports that
the different components of a Swam cluster listen on. You should use these to
configure your firewalls and other network access control lists.

- **Swarm manager.**
    - **Inbound 80/tcp (HTTP)**. This allows `docker pull` commands to work. If you plan to pull images from Docker Hub, you must allow Internet connections through port 80.
    - **Inbound 2375/tcp**. This allows Docker Engine CLI commands direct to the Engine daemon.
    - **Inbound 3375/tcp**. This allows Engine CLI commands to the Swarm manager.
    - **Inbound 22/tcp**. This allows remote management via SSH
- **Service Discovery**:
    - **Inbound 80/tcp (HTTP)**. This allows `docker pull` commands to work. If you plan to pull images from Docker Hub, you must allow Internet connections through port 80.
    - **Inbound *Discovery service port***. This needs setting to the port that the backend discovery service listens on (consul, etcd, or zookeeper).
    - **Inbound 22/tcp**. This allows remote management via SSH
- **Swarm nodes**:
    - **Inbound 80/tcp (HTTP)**. This allows `docker pull` commands to work. If you plan to pull images from Docker Hub, you must allow Internet connections through port 80.
    - **Inbound 2375/tcp**. This allows Engine CLI commands direct to the Docker daemon.
    - **Inbound 22/tcp**. This allows remote management via SSH.
- **Custom, cross-host container networks**:
    - **Inbound 7946/tcp** Allows for discovering other container networks.
    - **Inbound 7946/udp** Allows for discovering other container networks.
    - **Inbound <store-port>/tcp** Network key-value store service port.
    - **4789/udp** For the container overlay network.


If your firewalls and other network devices are connection state aware, they
will allow responses to established TCP connections. If your devices are not
state aware, you will need to open up ephemeral ports from 32768-65535. For
added security you can configure the ephemeral port rules to only allow
connections from interfaces on known Swarm devices.

If your Swarm cluster is configured for TLS, replace `2375` with `2376`, and
`3375` with `3376`.

The ports listed above are just for Swarm cluster operations  such as; cluster
creation, cluster management, and scheduling of containers against the cluster.
 You may need to open additional network ports for application-related
communications.

It is possible for different components of a Swarm cluster to exist on separate
networks. For example, many organizations operate separate management and
production networks. Some Docker Engine clients may exist on a management
network, while Swarm managers, discovery service instances, and nodes might
exist on one or more production networks. To offset against network failures,
you can deploy Swarm managers, discovery services, and nodes across multiple
production networks. In all of these cases you can use the list of ports above
to assist the work of your network infrastructure teams to efficiently and
securely configure your network.

## High Availability (HA)

All production environments should be highly available, meaning they are
continuously operational over long periods of time. To achieve high
availability, an environment must the survive failures of its individual
component parts.

The following sections discuss some technologies and best practices that can
enable you to build resilient, highly-available Swarm clusters. You can then use
these cluster to run your most demanding production applications and workloads.

### Swarm manager HA

The Swarm manager is responsible for accepting all commands coming in to a Swarm
cluster, and scheduling resources against the cluster. If the Swarm manager
becomes unavailable, some cluster operations cannot be performed until the Swarm
manager becomes available again. This is unacceptable in large-scale business
critical scenarios.

Swarm provides HA features to mitigate against possible failures of the Swarm
manager. You can use Swarm's HA feature to configure multiple Swarm managers for
a single cluster. These Swarm managers operate in an active/passive formation
with a single Swarm manager being the *primary*, and all others being
*secondaries*.

Swarm secondary managers operate as *warm standby's*, meaning they run in the
background of the primary Swarm manager. The secondary Swarm managers are online
and accept commands issued to the cluster, just as the primary Swarm manager.
However, any commands received by the secondaries are forwarded to the primary
where they are executed. Should the primary Swarm manager fail, a new primary is
elected from the surviving secondaries.

When creating HA Swarm managers, you should take care to distribute them over as
many *failure domains* as possible. A failure domain is a network section that
can be negatively affected if a critical device or service experiences problems.
For example, if your cluster is running in the Ireland Region of Amazon Web
Services (eu-west-1) and you configure three Swarm managers (1 x primary, 2 x
secondary), you should place one in each availability zone as shown below.

![](http://farm2.staticflickr.com/1657/24581727611_0a076b79de_b.jpg)

In this configuration, the Swarm cluster can survive the loss of any two
availability zones. For your applications to survive such failures, they must be
architected across as many failure domains as well.

For Swarm clusters serving high-demand, line-of-business applications, you
should have 3 or more Swarm managers. This configuration allows you to take one
manager down for maintenance, suffer an unexpected failure, and still continue
to manage and operate the cluster.

### Discovery service HA

The discovery service is a key component of a Swarm cluster. If the discovery
service becomes unavailable, this can prevent certain cluster operations. For
example, without a working discovery service, operations such as adding new
nodes to the cluster and making queries against the cluster configuration fail.
This is not acceptable in business critical production environments.

Swarm supports four backend discovery services:

- Hosted (not for production use)
- Consul
- etcd
- Zookeeper

Consul, etcd, and Zookeeper are all suitable for production, and should be
configured for high availability. You should use each service's existing tools
and best practices to configure these for HA.

For Swarm clusters serving high-demand, line-of-business applications, it is
recommended to have 5 or more discovery service instances. This due to the
replication/HA technologies they use (such as Paxos/Raft) requiring a strong
quorum. Having 5 instances allows you to take one down for maintenance, suffer
an unexpected failure, and still be able to achieve a strong quorum.

When creating a highly available Swarm discovery service, you should take care
to distribute each discovery service instance over as many failure domains as
possible. For example, if your cluster is running in the Ireland Region of
Amazon Web Services (eu-west-1) and you configure three discovery service
 instances, you should place one in each availability zone.

The diagram below shows a Swarm cluster configured for HA. It has three Swarm
managers and three discovery service instances spread over three failure
domains (availability zones). It also has Swarm nodes balanced across all three
 failure domains. The loss of two availability zones in the configuration shown
 below does not cause the Swarm cluster to go down.

![](http://farm2.staticflickr.com/1675/24380252320_999687d2bb_b.jpg)

It is possible to share the same Consul, etcd, or Zookeeper containers between
the Swarm discovery and Engine container networks. However, for best
performance and availability you should deploy dedicated instances &ndash; a
discovery instance for Swarm and another for your container networks.

### Multiple clouds

You can architect and build Swarm clusters that stretch across multiple cloud
providers, and even across public cloud and on premises infrastructures. The
diagram below shows an example Swarm cluster stretched across AWS and Azure.

![](http://farm2.staticflickr.com/1493/24676269945_d19daf856c_b.jpg)

While such architectures may appear to provide the ultimate in availability,
there are several factors to consider. Network latency can be problematic, as
can partitioning. As such, you should seriously consider technologies that
provide reliable, high speed, low latency connections into these cloud
platforms &ndash; technologies such as AWS Direct Connect and Azure
ExpressRoute.

If you are considering a production deployment across multiple infrastructures
like this, make sure you have good test coverage over your entire system.

### Isolated production environments

It is possible to run multiple environments, such as development, staging, and
production, on a single Swarm cluster. You accomplish this by tagging Swarm
nodes and using constraints to filter containers onto nodes tagged as
`production` or `staging` etc. However, this is not recommended. The recommended
approach is to air-gap production environments, especially high performance
business critical production environments.

For example, many companies not only deploy dedicated isolated infrastructures
for production &ndash; such as networks, storage, compute and other systems.
They also deploy separate management systems and policies. This results in
things like users having separate accounts for logging on to production systems
etc. In these types of environments, it is mandatory to deploy dedicated
production Swarm clusters that operate on the production hardware infrastructure
and follow thorough production management, monitoring, audit and other policies.

### Operating system selection

You should give careful consideration to the operating system that your Swarm
infrastructure relies on. This consideration is vital for production
environments.

It is not unusual for a company to use one operating system in development
environments, and a different one in production. A common example of this is to
use CentOS in development environments, but then to use Red Hat Enterprise Linux
(RHEL) in production. This decision is often a balance between cost and support.
CentOS Linux can be downloaded and used for free, but commercial support options
are few and far between. Whereas RHEL has an associated support and license
cost, but comes with world class commercial support from Red Hat.

When choosing the production operating system to use with your Swarm clusters,
you should choose one that closely matches what you have used in development and
staging environments. Although containers abstract much of the underlying OS,
some things are mandatory. For example, Docker container networks require Linux
kernel 3.16 or higher. Operating a 4.x kernel in development and staging and
then 3.14 in production will certainly cause issues.

You should also consider procedures and channels for deploying and potentially
patching your production operating systems.

## Performance

Performance is critical in environments that support business critical line of
business applications. The following sections discuss some technologies and
best practices that can help you build high performance Swarm clusters.

### Container networks

Docker Engine container networks are overlay networks and can be created across
multiple Engine hosts. For this reason, a container network requires a key-value
(KV) store to maintain network configuration and state. This KV store can be
shared in common with the one used by the Swarm cluster discovery service.
However, for best performance and fault isolation, you should deploy individual
KV store instances for container networks and Swarm discovery. This is
especially so in demanding business critical production environments.

Engine container networks also require version 3.16 or higher of the Linux
kernel. Higher kernel versions are usually preferred, but carry an increased
risk of instability because of the newness of the kernel. Where possible, you
should use a kernel version that is already approved for use in your production
environment. If you do not have a 3.16 or higher Linux kernel version approved
for production, you should begin the process of getting one as early as
possible.

### Scheduling strategies

<!-- NIGEL: This reads like an explanation of specific scheduling strategies rather than guidance on which strategy to pick for production or with consideration of a production architecture choice.  For example, is spread a problem in a multiple clouds or random not good for XXX type application for YYY reason?

Or perhaps there is nothing to consider when it comes to scheduling strategy and network / HA architecture, application, os choice etc. that good?

-->

Scheduling strategies are how Swarm decides which nodes on a cluster to start
containers on. Swarm supports the following strategies:

- spread
- binpack
- random (not for production use)

You can also write your own.

**Spread** is the default strategy. It attempts to balance the number of
containers evenly across all nodes in the cluster. This is a good choice for
high performance clusters, as it spreads container workload across all
resources in the cluster. These resources include CPU, RAM, storage, and
network bandwidth.

If your Swarm nodes are balanced across multiple failure domains, the spread
strategy evenly balance containers across those failure domains. However,
spread on its own is not aware of the roles of any of those containers, so has
no inteligence to spread multiple instances of the same service across failure
domains. To achieve this you should use tags and constraints.

The **binpack** strategy runs as many containers as possible on a node,
effectively filling it up, before scheduling containers on the next node.

This means that binpack does not use all cluster resources until the cluster
fills up. As a result, applications running on Swarm clusters that operate the
binpack strategy might not perform as well as those that operate the spread
strategy. However, binpack is a good choice for minimizing infrastructure
requirements and cost. For example, imagine you have a 10-node cluster where
each node has 16 CPUs and 128GB of RAM. However, your container workload across
 the entire cluster is only using the equivalent of 6 CPUs and 64GB RAM. The
spread strategy would balance containers across all nodes in the cluster.
However, the binpack strategy would fit all containers on a single node,
potentially allowing you turn off the additional nodes and save on cost.

## Ownership of Swarm clusters

The question of ownership is vital in production environments. It is therefore
vital that you consider and agree on all of the following when planning,
documenting, and deploying your production Swarm clusters.

- Whose budget does the production Swarm infrastructure come out of?
- Who owns the accounts that can administer and manage the production Swarm
cluster?
- Who is responsible for monitoring the production Swarm infrastructure?
- Who is responsible for patching and upgrading the production Swarm
infrastructure?
- On-call responsibilities and escalation procedures?

The above is not a complete list, and the answers to the questions will vary
depending on how your organization's and team's are structured. Some companies
are along way down the DevOps route, while others are not. Whatever situation
your company is in, it is important that you factor all of the above into the
planning, deployment, and ongoing management of your production Swarm clusters.


## Related information

* [Try Swarm at scale](swarm_at_scale/index.md)
* [Swarm and container networks](networking.md)
* [High availability in Docker Swarm](multi-manager-setup.md)
* [Universal Control plane](https://www.docker.com/products/docker-universal-control-plane)
