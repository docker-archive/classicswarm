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
[here](http://releases.k8s.io/release-1.0/examples/sharing-clusters/README.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Sharing Clusters

This example demonstrates how to access one kubernetes cluster from another. It only works if both clusters are running on the same network, on a cloud provider that provides a private ip range per network (eg: GCE, GKE, AWS).

## Setup

Create a cluster in US (you don't need to do this if you already have a running kubernetes cluster)

```shell
$ cluster/kube-up.sh
```

Before creating our second cluster, lets have a look at the kubectl config:

```yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: REDACTED
    server: https://104.197.84.16
  name: <clustername_us>
...
current-context: <clustername_us>
...
```

Now spin up the second cluster in Europe

```shell
$ ./cluster/kube-up.sh
$ KUBE_GCE_ZONE=europe-west1-b KUBE_GCE_INSTANCE_PREFIX=eu ./cluster/kube-up.sh
```

Your kubectl config should contain both clusters:

```yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: REDACTED
    server: https://146.148.25.221
  name: <clustername_eu>
- cluster:
    certificate-authority-data: REDACTED
    server: https://104.197.84.16
  name: <clustername_us>
...
current-context: kubernetesdev_eu
...
```

And kubectl get nodes should agree:

```
$ kubectl get nodes
NAME             LABELS                                  STATUS
eu-minion-0n61   kubernetes.io/hostname=eu-minion-0n61   Ready
eu-minion-79ua   kubernetes.io/hostname=eu-minion-79ua   Ready
eu-minion-7wz7   kubernetes.io/hostname=eu-minion-7wz7   Ready
eu-minion-loh2   kubernetes.io/hostname=eu-minion-loh2   Ready

$ kubectl config use-context <clustername_us>
$ kubectl get nodes
NAME                     LABELS                                                            STATUS
kubernetes-minion-5jtd   kubernetes.io/hostname=kubernetes-minion-5jtd                     Ready
kubernetes-minion-lqfc   kubernetes.io/hostname=kubernetes-minion-lqfc                     Ready
kubernetes-minion-sjra   kubernetes.io/hostname=kubernetes-minion-sjra                     Ready
kubernetes-minion-wul8   kubernetes.io/hostname=kubernetes-minion-wul8                     Ready
```

## Testing reachability

For this test to work we'll need to create a service in europe:

```
$ kubectl config use-context <clustername_eu>
$ kubectl create -f /tmp/secret.json
$ kubectl create -f examples/https-nginx/nginx-app.yaml
$ kubectl exec -it my-nginx-luiln -- echo "Europe nginx" >> /usr/share/nginx/html/index.html
$ kubectl get ep
NAME         ENDPOINTS
kubernetes   10.240.249.92:443
nginxsvc     10.244.0.4:80,10.244.0.4:443
```

Just to test reachability, we'll try hitting the Europe nginx from our initial US central cluster. Create a basic curl pod in the US cluster:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: curlpod
spec:
  containers:
  - image: radial/busyboxplus:curl
    command:
      - sleep
      - "360000000"
    imagePullPolicy: IfNotPresent
    name: curlcontainer
  restartPolicy: Always
```

And test that you can actually reach the test nginx service across continents

```
$ kubectl config use-context <clustername_us>
$ kubectl -it exec curlpod -- /bin/sh
[ root@curlpod:/ ]$ curl http://10.244.0.4:80
Europe nginx
```

## Granting access to the remote cluster

We will grant the US cluster access to the Europe cluster. Basically we're going to setup a secret that allows kubectl to function in a pod running in the US cluster, just like it did on our local machine in the previous step. First create a secret with the contents of the current .kube/config:

```shell
$ kubectl config use-context <clustername_eu>
$ go run ./make_secret.go --kubeconfig=$HOME/.kube/config > /tmp/secret.json
$ kubectl config use-context <clustername_us>
$ kubectl create -f /tmp/secret.json
```

Create a kubectl pod that uses the secret, in the US cluster.

```json
{
  "kind": "Pod",
  "apiVersion": "v1",
  "metadata": {
    "name": "kubectl-tester"
  },
  "spec": {
    "volumes": [
       {
            "name": "secret-volume",
            "secret": {
                "secretName": "kubeconfig"
            }
        }
    ],
    "containers": [
      {
        "name": "kubectl",
        "image": "bprashanth/kubectl:0.0",
        "imagePullPolicy": "Always",
        "env": [
            {
                "name": "KUBECONFIG",
                "value": "/.kube/config"
            }
        ],
        "args": [
          "proxy", "-p", "8001"
        ],
        "volumeMounts": [
          {
              "name": "secret-volume",
               "mountPath": "/.kube"
          }
        ]
      }
    ]
  }
}
```

And check that you can access the remote cluster

```shell
$ kubectl config use-context <clustername_us>
$ kubectl exec -it kubectl-tester bash

kubectl-tester $ kubectl get nodes
NAME             LABELS                                  STATUS
eu-minion-0n61   kubernetes.io/hostname=eu-minion-0n61   Ready
eu-minion-79ua   kubernetes.io/hostname=eu-minion-79ua   Ready
eu-minion-7wz7   kubernetes.io/hostname=eu-minion-7wz7   Ready
eu-minion-loh2   kubernetes.io/hostname=eu-minion-loh2   Ready
```

For a more advanced example of sharing clusters, see the [service-loadbalancer](https://github.com/kubernetes/contrib/tree/master/service-loadbalancer/README.md)


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/examples/sharing-clusters/README.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
