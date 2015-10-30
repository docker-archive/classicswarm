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
[here](http://releases.k8s.io/release-1.0/docs/user-guide/connecting-to-applications-proxy.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Connecting to applications: kubectl proxy and apiserver proxy

You have seen the [basics](accessing-the-cluster.md) about `kubectl proxy` and `apiserver proxy`. This guide shows how to use them together to access a service([kube-ui](ui.md)) running on the Kubernetes cluster from your workstation.


## Getting the apiserver proxy URL of kube-ui

kube-ui is deployed as a cluster add-on. To find its apiserver proxy URL,

```console
$ kubectl cluster-info | grep "KubeUI"
KubeUI is running at https://173.255.119.104/api/v1/proxy/namespaces/kube-system/services/kube-ui
```

if this command does not find the URL, try the steps [here](ui.md#accessing-the-ui).


## Connecting to the kube-ui service from your local workstation

The above proxy URL is an access to the kube-ui service provided by the apiserver. To access it, you still need to authenticate to the apiserver. `kubectl proxy` can handle the authentication.

```console
$ kubectl proxy --port=8001
Starting to serve on localhost:8001
```

Now you can access the kube-ui service on your local workstation at [http://localhost:8001/api/v1/proxy/namespaces/kube-system/services/kube-ui](http://localhost:8001/api/v1/proxy/namespaces/kube-system/services/kube-ui)


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/user-guide/connecting-to-applications-proxy.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
