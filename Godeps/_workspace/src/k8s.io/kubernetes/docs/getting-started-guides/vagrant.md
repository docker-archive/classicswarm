<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/getting-started-guides/vagrant.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

## Getting started with Vagrant

Running Kubernetes with Vagrant (and VirtualBox) is an easy way to run/test/develop on your local machine (Linux, Mac OS X).

**Table of Contents**

- [Prerequisites](#prerequisites)
- [Setup](#setup)
- [Interacting with your Kubernetes cluster with Vagrant.](#interacting-with-your-kubernetes-cluster-with-vagrant)
- [Authenticating with your master](#authenticating-with-your-master)
- [Running containers](#running-containers)
- [Troubleshooting](#troubleshooting)
    - [I keep downloading the same (large) box all the time!](#i-keep-downloading-the-same-large-box-all-the-time)
    - [I just created the cluster, but I am getting authorization errors!](#i-just-created-the-cluster-but-i-am-getting-authorization-errors)
    - [I just created the cluster, but I do not see my container running!](#i-just-created-the-cluster-but-i-do-not-see-my-container-running)
    - [I want to make changes to Kubernetes code!](#i-want-to-make-changes-to-kubernetes-code)
    - [I have brought Vagrant up but the nodes cannot validate!](#i-have-brought-vagrant-up-but-the-nodes-cannot-validate)
    - [I want to change the number of nodes!](#i-want-to-change-the-number-of-nodes)
    - [I want my VMs to have more memory!](#i-want-my-vms-to-have-more-memory)
    - [I ran vagrant suspend and nothing works!](#i-ran-vagrant-suspend-and-nothing-works)
    - [I want vagrant to sync folders via nfs!](#i-want-vagrant-to-sync-folders-via-nfs)

### Prerequisites

1. Install latest version >= 1.6.2 of vagrant from http://www.vagrantup.com/downloads.html
2. Install one of:
   1. The latest version of Virtual Box from https://www.virtualbox.org/wiki/Downloads
   2. [VMWare Fusion](https://www.vmware.com/products/fusion/) version 5 or greater as well as the appropriate [Vagrant VMWare Fusion provider](https://www.vagrantup.com/vmware)
   3. [VMWare Workstation](https://www.vmware.com/products/workstation/) version 9 or greater as well as the [Vagrant VMWare Workstation provider](https://www.vagrantup.com/vmware)
   4. [Parallels Desktop](https://www.parallels.com/products/desktop/) version 9 or greater as well as the [Vagrant Parallels provider](https://parallels.github.io/vagrant-parallels/)
   5. libvirt with KVM and enable support of hardware virtualisation. [Vagrant-libvirt](https://github.com/pradels/vagrant-libvirt). For fedora provided official rpm, and possible to use `yum install vagrant-libvirt`

### Setup

Setting up a cluster is as simple as running:

```sh
export KUBERNETES_PROVIDER=vagrant
curl -sS https://get.k8s.io | bash
```

Alternatively, you can download [Kubernetes release](https://github.com/kubernetes/kubernetes/releases) and extract the archive. To start your local cluster, open a shell and run:

```sh
cd kubernetes

export KUBERNETES_PROVIDER=vagrant
./cluster/kube-up.sh
```

The `KUBERNETES_PROVIDER` environment variable tells all of the various cluster management scripts which variant to use.  If you forget to set this, the assumption is you are running on Google Compute Engine.

By default, the Vagrant setup will create a single master VM (called kubernetes-master) and one node (called kubernetes-minion-1). Each VM will take 1 GB, so make sure you have at least 2GB to 4GB of free memory (plus appropriate free disk space).

Vagrant will provision each machine in the cluster with all the necessary components to run Kubernetes.  The initial setup can take a few minutes to complete on each machine.

If you installed more than one Vagrant provider, Kubernetes will usually pick the appropriate one. However, you can override which one Kubernetes will use by setting the [`VAGRANT_DEFAULT_PROVIDER`](https://docs.vagrantup.com/v2/providers/default.html) environment variable:

```sh
export VAGRANT_DEFAULT_PROVIDER=parallels
export KUBERNETES_PROVIDER=vagrant
./cluster/kube-up.sh
```

By default, each VM in the cluster is running Fedora.

To access the master or any node:

```sh
vagrant ssh master
vagrant ssh minion-1
```

If you are running more than one node, you can access the others by:

```sh
vagrant ssh minion-2
vagrant ssh minion-3
```

Each node in the cluster installs the docker daemon and the kubelet.

The master node instantiates the Kubernetes master components as pods on the machine.

To view the service status and/or logs on the kubernetes-master:

```console
[vagrant@kubernetes-master ~] $ vagrant ssh master
[vagrant@kubernetes-master ~] $ sudo su

[root@kubernetes-master ~] $ systemctl status kubelet
[root@kubernetes-master ~] $ journalctl -ru kubelet

[root@kubernetes-master ~] $ systemctl status docker
[root@kubernetes-master ~] $ journalctl -ru docker

[root@kubernetes-master ~] $ tail -f /var/log/kube-apiserver.log
[root@kubernetes-master ~] $ tail -f /var/log/kube-controller-manager.log
[root@kubernetes-master ~] $ tail -f /var/log/kube-scheduler.log
```

To view the services on any of the nodes:

```console
[vagrant@kubernetes-master ~] $ vagrant ssh minion-1
[vagrant@kubernetes-master ~] $ sudo su

[root@kubernetes-master ~] $ systemctl status kubelet
[root@kubernetes-master ~] $ journalctl -ru kubelet

[root@kubernetes-master ~] $ systemctl status docker
[root@kubernetes-master ~] $ journalctl -ru docker
```

### Interacting with your Kubernetes cluster with Vagrant.

With your Kubernetes cluster up, you can manage the nodes in your cluster with the regular Vagrant commands.

To push updates to new Kubernetes code after making source changes:

```sh
./cluster/kube-push.sh
```

To stop and then restart the cluster:

```sh
vagrant halt
./cluster/kube-up.sh
```

To destroy the cluster:

```sh
vagrant destroy
```

Once your Vagrant machines are up and provisioned, the first thing to do is to check that you can use the `kubectl.sh` script.

You may need to build the binaries first, you can do this with `make`

```console
$ ./cluster/kubectl.sh get nodes

NAME                LABELS
10.245.1.4          <none>
10.245.1.5          <none>
10.245.1.3          <none>
```

### Authenticating with your master

When using the vagrant provider in Kubernetes, the `cluster/kubectl.sh` script will cache your credentials in a `~/.kubernetes_vagrant_auth` file so you will not be prompted for them in the future.

```sh
cat ~/.kubernetes_vagrant_auth
```

```json
{ "User": "vagrant",
  "Password": "vagrant",
  "CAFile": "/home/k8s_user/.kubernetes.vagrant.ca.crt",
  "CertFile": "/home/k8s_user/.kubecfg.vagrant.crt",
  "KeyFile": "/home/k8s_user/.kubecfg.vagrant.key"
}
```

You should now be set to use the `cluster/kubectl.sh` script. For example try to list the nodes that you have started with:

```sh
./cluster/kubectl.sh get nodes
```

### Running containers

Your cluster is running, you can list the nodes in your cluster:

```console
$ ./cluster/kubectl.sh get nodes

NAME                 LABELS
10.245.2.4           <none>
10.245.2.3           <none>
10.245.2.2           <none>
```

Now start running some containers!

You can now use any of the `cluster/kube-*.sh` commands to interact with your VM machines.
Before starting a container there will be no pods, services and replication controllers.

```console
$ ./cluster/kubectl.sh get pods
NAME        READY     STATUS    RESTARTS   AGE

$ ./cluster/kubectl.sh get services
NAME              CLUSTER_IP       EXTERNAL_IP       PORT(S)       SELECTOR               AGE

$ ./cluster/kubectl.sh get replicationcontrollers
CONTROLLER   CONTAINER(S)   IMAGE(S)   SELECTOR   REPLICAS
```

Start a container running nginx with a replication controller and three replicas

```console
$ ./cluster/kubectl.sh run my-nginx --image=nginx --replicas=3 --port=80
```

When listing the pods, you will see that three containers have been started and are in Waiting state:

```console
$ ./cluster/kubectl.sh get pods
NAME             READY     STATUS    RESTARTS   AGE
my-nginx-5kq0g   0/1       Pending   0          10s
my-nginx-gr3hh   0/1       Pending   0          10s
my-nginx-xql4j   0/1       Pending   0          10s
```

You need to wait for the provisioning to complete, you can monitor the nodes by doing:

```console
$ vagrant ssh minion-1 -c 'sudo docker images'
kubernetes-minion-1:
    REPOSITORY          TAG                 IMAGE ID            CREATED             VIRTUAL SIZE
    <none>              <none>              96864a7d2df3        26 hours ago        204.4 MB
    google/cadvisor     latest              e0575e677c50        13 days ago         12.64 MB
    kubernetes/pause    latest              6c4579af347b        8 weeks ago         239.8 kB
```

Once the docker image for nginx has been downloaded, the container will start and you can list it:

```console
$ vagrant ssh minion-1 -c 'sudo docker ps'
kubernetes-minion-1:
    CONTAINER ID        IMAGE                     COMMAND                CREATED             STATUS              PORTS                    NAMES
    dbe79bf6e25b        nginx:latest              "nginx"                21 seconds ago      Up 19 seconds                                k8s--mynginx.8c5b8a3a--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1.etcd--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1--fcfa837f
    fa0e29c94501        kubernetes/pause:latest   "/pause"               8 minutes ago       Up 8 minutes        0.0.0.0:8080->80/tcp     k8s--net.a90e7ce4--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1.etcd--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1--baf5b21b
    aa2ee3ed844a        google/cadvisor:latest    "/usr/bin/cadvisor"    38 minutes ago      Up 38 minutes                                k8s--cadvisor.9e90d182--cadvisor_-_agent.file--4626b3a2
    65a3a926f357        kubernetes/pause:latest   "/pause"               39 minutes ago      Up 39 minutes       0.0.0.0:4194->8080/tcp   k8s--net.c5ba7f0e--cadvisor_-_agent.file--342fd561
```

Going back to listing the pods, services and replicationcontrollers, you now have:

```console
$ ./cluster/kubectl.sh get pods
NAME             READY     STATUS    RESTARTS   AGE
my-nginx-5kq0g   1/1       Running   0          1m
my-nginx-gr3hh   1/1       Running   0          1m
my-nginx-xql4j   1/1       Running   0          1m

$ ./cluster/kubectl.sh get services
NAME              CLUSTER_IP       EXTERNAL_IP       PORT(S)       SELECTOR               AGE
my-nginx          10.0.0.1         <none>            80/TCP        run=my-nginx           1h
```

We did not start any services, hence there are none listed. But we see three replicas displayed properly.
Check the [guestbook](../../examples/guestbook/README.md) application to learn how to create a service.
You can already play with scaling the replicas with:

```console
$ ./cluster/kubectl.sh scale rc my-nginx --replicas=2
$ ./cluster/kubectl.sh get pods
NAME             READY     STATUS    RESTARTS   AGE
my-nginx-5kq0g   1/1       Running   0          2m
my-nginx-gr3hh   1/1       Running   0          2m
```

Congratulations!

### Troubleshooting

#### I keep downloading the same (large) box all the time!

By default the Vagrantfile will download the box from S3. You can change this (and cache the box locally) by providing a name and an alternate URL when calling `kube-up.sh`

```sh
export KUBERNETES_BOX_NAME=choose_your_own_name_for_your_kuber_box
export KUBERNETES_BOX_URL=path_of_your_kuber_box
export KUBERNETES_PROVIDER=vagrant
./cluster/kube-up.sh
```

#### I just created the cluster, but I am getting authorization errors!

You probably have an incorrect ~/.kubernetes_vagrant_auth file for the cluster you are attempting to contact.

```sh
rm ~/.kubernetes_vagrant_auth
```

After using kubectl.sh make sure that the correct credentials are set:

```sh
cat ~/.kubernetes_vagrant_auth
```

```json
{
  "User": "vagrant",
  "Password": "vagrant"
}
```

#### I just created the cluster, but I do not see my container running!

If this is your first time creating the cluster, the kubelet on each node schedules a number of docker pull requests to fetch prerequisite images.  This can take some time and as a result may delay your initial pod getting provisioned.

#### I want to make changes to Kubernetes code!

To set up a vagrant cluster for hacking, follow the [vagrant developer guide](../devel/developer-guides/vagrant.md).

#### I have brought Vagrant up but the nodes cannot validate!

Log on to one of the nodes (`vagrant ssh minion-1`) and inspect the salt minion log (`sudo cat /var/log/salt/minion`).

#### I want to change the number of nodes!

You can control the number of nodes that are instantiated via the environment variable `NUM_MINIONS` on your host machine.  If you plan to work with replicas, we strongly encourage you to work with enough nodes to satisfy your largest intended replica size.  If you do not plan to work with replicas, you can save some system resources by running with a single node. You do this, by setting `NUM_MINIONS` to 1 like so:

```sh
export NUM_MINIONS=1
```

#### I want my VMs to have more memory!

You can control the memory allotted to virtual machines with the `KUBERNETES_MEMORY` environment variable.
Just set it to the number of megabytes you would like the machines to have. For example:

```sh
export KUBERNETES_MEMORY=2048
```

If you need more granular control, you can set the amount of memory for the master and nodes independently. For example:

```sh
export KUBERNETES_MASTER_MEMORY=1536
export KUBERNETES_MINION_MEMORY=2048
```

#### I ran vagrant suspend and nothing works!

`vagrant suspend` seems to mess up the network.  This is not supported at this time.

#### I want vagrant to sync folders via nfs!

You can ensure that vagrant uses nfs to sync folders with virtual machines by setting the KUBERNETES_VAGRANT_USE_NFS environment variable to 'true'. nfs is faster than virtualbox or vmware's 'shared folders' and does not require guest additions. See the [vagrant docs](http://docs.vagrantup.com/v2/synced-folders/nfs.html) for details on configuring nfs on the host. This setting will have no effect on the libvirt provider, which uses nfs by default. For example:

```sh
export KUBERNETES_VAGRANT_USE_NFS=true
```


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/getting-started-guides/vagrant.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
