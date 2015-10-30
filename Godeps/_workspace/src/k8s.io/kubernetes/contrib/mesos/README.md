# Kubernetes-Mesos

Kubernetes-Mesos modifies Kubernetes to act as an [Apache Mesos](http://mesos.apache.org/) framework.

## Features On Mesos

Kubernetes gains the following benefits when installed on Mesos:

- **Node-Level Auto-Scaling** - Kubernetes minion nodes are created automatically, up to the size of the provisioned Mesos cluster.
- **Resource Sharing** - Co-location of Kubernetes with other popular next-generation services on the same cluster (e.g. [Hadoop](https://github.com/mesos/hadoop), [Spark](http://spark.apache.org/), and [Chronos](https://mesos.github.io/chronos/), [Cassandra](http://mesosphere.github.io/cassandra-mesos/), etc.). Resources are allocated to the frameworks based on fairness and can be claimed or passed on depending on framework load.
- **Independence from special Network Infrastructure** - Mesos can (but of course doesn't have to) run on networks which cannot assign a routable IP to every container. The Kubernetes on Mesos endpoint controller is specially modified to allow pods to communicate with services in such an environment.

## Features On DCOS

Kubernetes can also be installed on [Mesosphere DCOS](https://mesosphere.com/learn/), which runs Mesos as its core. This provides the following *additional* enterprise features:

- **High Availability** - Kubernetes components themselves run within Marathon, which manages restarting/recreating them if they fail, even on a different host if the original host might fail completely.
- **Easy Installation** - One-step installation via the [DCOS CLI](https://github.com/mesosphere/dcos-cli) or DCOS UI. Both download releases from the [Mesosphere Universe](https://github.com/mesosphere/universe), [Multiverse](https://github.com/mesosphere/multiverse), or private package repositories.
- **Easy Maintenance** - See what's going on in the cluster with the DCOS UI.

For more information about how Kubernetes-Mesos is different from Kubernetes, see [Architecture](./docs/architecture.md).


## Release Status

Kubernetes-Mesos is alpha quality, still under active development, and not yet recommended for production systems.

For more information about development progress, see the [known issues](./docs/issues.md) or the [kubernetes-mesos repository](https://github.com/mesosphere/kubernetes-mesos) where backlog issues are tracked.

## Usage

This project combines concepts and technologies from two already-complex projects: Mesos and Kubernetes. It may help to familiarize yourself with the basics of each project before reading on:

* [Mesos Documentation](http://mesos.apache.org/documentation/latest)
* [Kubernetes Documentation](../../README.md)

To get up and running with Kubernetes-Mesos, follow the [Getting started guide](../../docs/getting-started-guides/mesos.md).


[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/contrib/mesos/README.md?pixel)]()
