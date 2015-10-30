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
[here](http://releases.k8s.io/release-1.0/docs/user-guide/compute-resources.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Compute Resources

**Table of Contents**
<!-- BEGIN MUNGE: GENERATED_TOC -->

- [Compute Resources](#compute-resources)
  - [Resource Requests and Limits of Pod and Container](#resource-requests-and-limits-of-pod-and-container)
  - [How Pods with Resource Requests are Scheduled](#how-pods-with-resource-requests-are-scheduled)
  - [How Pods with Resource Limits are Run](#how-pods-with-resource-limits-are-run)
  - [Monitoring Compute Resource Usage](#monitoring-compute-resource-usage)
  - [Troubleshooting](#troubleshooting)
    - [My pods are pending with event message failedScheduling](#my-pods-are-pending-with-event-message-failedscheduling)
    - [My container is terminated](#my-container-is-terminated)
  - [Planned Improvements](#planned-improvements)

<!-- END MUNGE: GENERATED_TOC -->

When specifying a [pod](pods.md), you can optionally specify how much CPU and memory (RAM) each
container needs.  When containers have their resource requests specified, the scheduler is
able to make better decisions about which nodes to place pods on; and when containers have their
limits specified, contention for resources on a node can be handled in a specified manner. For
more details about the difference between requests and limits, please refer to
[Resource QoS](../proposals/resource-qos.md).

*CPU* and *memory* are each a *resource type*.  A resource type has a base unit.  CPU is specified
in units of cores.  Memory is specified in units of bytes.

CPU and RAM are collectively referred to as *compute resources*, or just *resources*.  Compute
resources are measureable quantities which can be requested, allocated, and consumed.  They are
distinct from [API resources](working-with-resources.md).  API resources, such as pods and
[services](services.md) are objects that can be written to and retrieved from the Kubernetes API
server.

## Resource Requests and Limits of Pod and Container

Each container of a Pod can optionally specify `spec.container[].resources.limits.cpu` and/or
`spec.container[].resources.limits.memory` and/or `spec.container[].resources.requests.cpu`
and/or `spec.container[].resources.requests.memory`.

Specifying resource requests and/or limits is optional. In some clusters, unset limits or requests
may be replaced with default values when a pod is created or updated. The default value depends on
how the cluster is configured. If value of requests is not specified, they are set to be equal
to limits by default. Please note that resource limits must be greater than or equal to resource
requests.

Although requests/limits can only be specified on individual containers, it is convenient to talk
about pod resource requests/limits.  A *pod resource request/limit* for a particular resource
type is the sum of the resource requests/limits of that type for each container in the pod, with
unset values treated as zero (or equal to default values in some cluster configurations).

The following pod has two containers.  Each has a request of 0.25 core of cpu and 64MiB
(2<sup>20</sup> bytes) of memory and a limit of 0.5 core of cpu and 128MiB of memory. The pod can
be said to have a request of 0.5 core and 128 MiB of memory and a limit of 1 core and 256MiB of
memory.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: frontend
spec:
  containers:
  - name: db
    image: mysql
    resources:
	  requests:
	    memory: "64Mi"
		cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
  - name: wp
    image: wordpress
    resources:
      requests:
	    memory: "64Mi"
		cpu: "250m"
	  limits:
        memory: "128Mi"
        cpu: "500m"
```

## How Pods with Resource Requests are Scheduled

When a pod is created, the Kubernetes scheduler selects a node for the pod to
run on.  Each node has a maximum capacity for each of the resource types: the
amount of CPU and memory it can provide for pods.  The scheduler ensures that,
for each resource type (CPU and memory), the sum of the resource requests of the
containers scheduled to the node is less than the capacity of the node.  Note
that although actual memory or CPU resource usage on nodes is very low, the
scheduler will still refuse to place pods onto nodes if the capacity check
fails.  This protects against a resource shortage on a node when resource usage
later increases, such as due to a daily peak in request rate.

## How Pods with Resource Limits are Run

When kubelet starts a container of a pod, it passes the CPU and memory limits to the container
runner (Docker or rkt).

When using Docker:
- The `spec.container[].resources.limits.cpu` is multiplied by 1024, converted to an integer, and
  used as the value of the [`--cpu-shares`](
  https://docs.docker.com/reference/run/#runtime-constraints-on-resources) flag to the `docker run`
  command.
- The `spec.container[].resources.limits.memory` is converted to an integer, and used as the value
  of the [`--memory`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) flag
  to the `docker run` command.

**TODO: document behavior for rkt**

If a container exceeds its memory limit, it may be terminated.  If it is restartable, it will be
restarted by kubelet, as will any other type of runtime failure.

A container may or may not be allowed to exceed its CPU limit for extended periods of time.
However, it will not be killed for excessive CPU usage.

To determine if a container cannot be scheduled or is being killed due to resource limits, see the
"Troubleshooting" section below.

## Monitoring Compute Resource Usage

The resource usage of a pod is reported as part of the Pod status.

If [optional monitoring](http://releases.k8s.io/HEAD/cluster/addons/cluster-monitoring/README.md) is configured for your cluster,
then pod resource usage can be retrieved from the monitoring system.

## Troubleshooting

### My pods are pending with event message failedScheduling

If the scheduler cannot find any node where a pod can fit, then the pod will remain unscheduled
until a place can be found.    An event will be produced each time the scheduler fails to find a
place for the pod, like this:

```console
$ kubectl describe pod frontend | grep -A 3 Events
Events:
  FirstSeen	LastSeen	 Count	From          Subobject   PathReason			Message
  36s		5s		 6	    {scheduler }              FailedScheduling	Failed for reason PodExceedsFreeCPU and possibly others

```

In the case shown above, the pod "frontend" fails to be scheduled due to insufficient
CPU resource on the node. Similar error messages can also suggest failure due to insufficient
memory (PodExceedsFreeMemory). In general, if a pod or pods are pending with this message and
alike, then there are several things to try:
- Add more nodes to the cluster.
- Terminate unneeded pods to make room for pending pods.
- Check that the pod is not larger than all the nodes.  For example, if all the nodes
have a capacity of `cpu: 1`, then a pod with a limit of `cpu: 1.1` will never be scheduled.

You can check node capacities and amounts allocated with the `kubectl describe nodes` command.
For example:

```console
$ kubectl describe nodes gke-cluster-4-386701dd-node-ww4p
Name:			gke-cluster-4-386701dd-node-ww4p
[ ... lines removed for clarity ...]
Capacity:
 cpu:		1
 memory:	464Mi
 pods:		40
Allocated resources (total requests):
 cpu:		910m
 memory:	2370Mi
 pods:		4
[ ... lines removed for clarity ...]
Pods:				(4 in total)
  Namespace			Name								CPU(milliCPU)			Memory(bytes)
  frontend 			webserver-ffj8j							500 (50% of total)		2097152000 (50% of total)
  kube-system			fluentd-cloud-logging-gke-cluster-4-386701dd-node-ww4p		100 (10% of total)		209715200 (5% of total)
  kube-system			kube-dns-v8-qopgw						310 (31% of total)		178257920 (4% of total)
TotalResourceLimits:
  CPU(milliCPU):		910 (91% of total)
  Memory(bytes):		2485125120 (59% of total)
[ ... lines removed for clarity ...]
```

Here you can see from the `Allocated resources` section that that a pod which ask for more than
90 millicpus or more than 1341MiB of memory will not be able to fit on this node.

Looking at the `Pods` section, you can see which pods are taking up space on the node.

The [resource quota](../admin/resource-quota.md) feature can be configured
to limit the total amount of resources that can be consumed.  If used in conjunction
with namespaces, it can prevent one team from hogging all the resources.

### My container is terminated

Your container may be terminated because it's resource-starved. To check if a container is being killed because it is hitting a resource limit, call `kubectl describe pod`
on the pod you are interested in:

```console
[12:54:41] $ ./cluster/kubectl.sh describe pod simmemleak-hra99
Name:                           simmemleak-hra99
Namespace:                      default
Image(s):                       saadali/simmemleak
Node:                           kubernetes-minion-tf0f/10.240.216.66
Labels:                         name=simmemleak
Status:                         Running
Reason:             
Message:            
IP:                             10.244.2.75
Replication Controllers:        simmemleak (1/1 replicas created)
Containers:
  simmemleak:
    Image:  saadali/simmemleak
    Limits:
      cpu:                      100m
      memory:                   50Mi
    State:                      Running
      Started:                  Tue, 07 Jul 2015 12:54:41 -0700
    Last Termination State:     Terminated
      Exit Code:                1
      Started:                  Fri, 07 Jul 2015 12:54:30 -0700
      Finished:                 Fri, 07 Jul 2015 12:54:33 -0700
    Ready:                      False
    Restart Count:              5
Conditions:
  Type      Status
  Ready     False 
Events:
  FirstSeen                         LastSeen                         Count  From                              SubobjectPath                       Reason      Message
  Tue, 07 Jul 2015 12:53:51 -0700   Tue, 07 Jul 2015 12:53:51 -0700  1      {scheduler }                                                          scheduled   Successfully assigned simmemleak-hra99 to kubernetes-minion-tf0f
  Tue, 07 Jul 2015 12:53:51 -0700   Tue, 07 Jul 2015 12:53:51 -0700  1      {kubelet kubernetes-minion-tf0f}  implicitly required container POD   pulled      Pod container image "gcr.io/google_containers/pause:0.8.0" already present on machine
  Tue, 07 Jul 2015 12:53:51 -0700   Tue, 07 Jul 2015 12:53:51 -0700  1      {kubelet kubernetes-minion-tf0f}  implicitly required container POD   created     Created with docker id 6a41280f516d
  Tue, 07 Jul 2015 12:53:51 -0700   Tue, 07 Jul 2015 12:53:51 -0700  1      {kubelet kubernetes-minion-tf0f}  implicitly required container POD   started     Started with docker id 6a41280f516d
  Tue, 07 Jul 2015 12:53:51 -0700   Tue, 07 Jul 2015 12:53:51 -0700  1      {kubelet kubernetes-minion-tf0f}  spec.containers{simmemleak}         created     Created with docker id 87348f12526a
```

The `Restart Count:  5` indicates that the `simmemleak` container in this pod was terminated and restarted 5 times.

You can call `get pod` with the `-o go-template=...` option to fetch the status of previously terminated containers:

```console
[13:59:01] $ ./cluster/kubectl.sh  get pod -o go-template='{{range.status.containerStatuses}}{{"Container Name: "}}{{.name}}{{"\r\nLastState: "}}{{.lastState}}{{end}}'  simmemleak-60xbc
Container Name: simmemleak
LastState: map[terminated:map[exitCode:137 reason:OOM Killed startedAt:2015-07-07T20:58:43Z finishedAt:2015-07-07T20:58:43Z containerID:docker://0e4095bba1feccdfe7ef9fb6ebffe972b4b14285d5acdec6f0d3ae8a22fad8b2]][13:59:03] clusterScaleDoc ~/go/src/github.com/kubernetes/kubernetes $ 
```

We can see that this container was terminated because `reason:OOM Killed`, where *OOM* stands for Out Of Memory.

## Planned Improvements

The current system only allows resource quantities to be specified on a container.
It is planned to improve accounting for resources which are shared by all containers in a pod,
such as [EmptyDir volumes](volumes.md#emptydir).

The current system only supports container requests and limits for CPU and Memory.
It is planned to add new resource types, including a node disk space
resource, and a framework for adding custom [resource types](../design/resources.md#resource-types).

Kubernetes supports overcommitment of resources by supporting multiple levels of [Quality of Service](http://issue.k8s.io/168).

Currently, one unit of CPU means different things on different cloud providers, and on different
machine types within the same cloud providers.  For example, on AWS, the capacity of a node
is reported in [ECUs](http://aws.amazon.com/ec2/faqs/), while in GCE it is reported in logical
cores.  We plan to revise the definition of the cpu resource to allow for more consistency
across providers and platforms.


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/user-guide/compute-resources.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
