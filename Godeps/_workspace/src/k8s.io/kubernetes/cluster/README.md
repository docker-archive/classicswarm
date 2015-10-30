# Cluster Configuration

The scripts and data in this directory automate creation and configuration of a Kubernetes cluster, including networking, DNS, nodes, and master components.

See the [getting-started guides](../docs/getting-started-guides) for examples of how to use the scripts.

*cloudprovider*/`config-default.sh` contains a set of tweakable definitions/parameters for the cluster.

The heavy lifting of configuring the VMs is done by [SaltStack](http://www.saltstack.com/).


[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/cluster/README.md?pixel)]()
