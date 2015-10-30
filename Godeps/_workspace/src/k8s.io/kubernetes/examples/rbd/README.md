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
[here](http://releases.k8s.io/release-1.0/examples/rbd/README.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# How to Use it?

Install Ceph on the Kubernetes host. For example, on Fedora 21

    # yum -y install ceph

If you don't have a Ceph cluster, you can set up a [containerized Ceph cluster](https://github.com/rootfs/docker-ceph)

Then get the keyring from the Ceph cluster and copy it to */etc/ceph/keyring*.

Once you have installed Ceph and new Kubernetes, you can create a pod based on my examples [rbd.json](rbd.json)  [rbd-with-secret.json](rbd-with-secret.json). In the pod JSON, you need to provide the following information.

- *monitors*:  Ceph monitors.
- *pool*: The name of the RADOS pool, if not provided, default *rbd* pool is used.
- *image*: The image name that rbd has created.
- *user*: The RADOS user name. If not provided, default *admin* is used.
- *keyring*: The path to the keyring file. If not provided, default */etc/ceph/keyring* is used.
- *secretName*: The name of the authentication secrets. If provided, *secretName* overrides *keyring*. Note, see below about how to create a secret.
- *fsType*: The filesystem type (ext4, xfs, etc) that formatted on the device.
- *readOnly*: Whether the filesystem is used as readOnly.

# Use Ceph Authentication Secret

If Ceph authentication secret is provided, the secret should be first be base64 encoded, then encoded string is placed in a secret yaml. An example yaml is provided [here](secret/ceph-secret.yaml). Then post the secret through ```kubectl``` in the following command.

```console
    # kubectl create -f examples/rbd/secret/ceph-secret.yaml
```

# Get started

Here are my commands:

```console
    # kubectl create -f examples/rbd/rbd.json
    # kubectl get pods
```

On the Kubernetes host, I got these in mount output

```console
    #mount |grep kub
	/dev/rbd0 on /var/lib/kubelet/plugins/kubernetes.io/rbd/rbd/kube-image-foo type ext4 (ro,relatime,stripe=4096,data=ordered)
	/dev/rbd0 on /var/lib/kubelet/pods/ec2166b4-de07-11e4-aaf5-d4bed9b39058/volumes/kubernetes.io~rbd/rbdpd type ext4 (ro,relatime,stripe=4096,data=ordered)
```

 If you ssh to that machine, you can run `docker ps` to see the actual pod and `docker inspect` to see the volumes used by the container.


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/examples/rbd/README.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
