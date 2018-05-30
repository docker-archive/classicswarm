# Use Mesos Revocable Resource in Swarm

## Introduction

Revocable resources was introduced into Mesos with the project [MESOS-354](https://issues.apache.org/jira/browse/MESOS-354) , so currently Mesos can offer two types of resources to the frameworks running on top of it:

 - **Regular resource** (or non-revocable resource): These are the resources offered by Mesos by default, once used to launch tasks, this kind of resources will not be revoked by Mesos.
 - **Revocable resource**: These are the resources not fully utilized by a Mesos framework, such resources can be offered by Mesos to another framework as revocable resources, any tasks launched using these resources could get preempted or throttled at any time. This could be used by frameworks to run best effort tasks that do not need strict uptime or performance guarantees.

Currently Swarm can only receive regular resources from Mesos. In this proposal, we'd like enable Swarm to also receive revocable resources from Mesos, and enable Swarm users to choose type of resources to create their containers, e.g., create high priority container with regular resources and low priority container with revocable resources. In this way, when there are other frameworks running on Mesos along with Swarm, the underutilized resources of other frameworks can be used by Swarm so that the overall resource utilization of the Mesos cluster can be improved.

## Use cases

 1. As a Swarm cluster operator, I want to enable Swarm to receive revocable resources from Mesos.
 2. As a Swarm user, I want to create my high priority container with regular resources only.
 3. As a Swarm user, I want to create my medium priority container with regular resources first, if there are no enough regular resources, it is OK to create the container with revocable resources.
 4. As a Swarm user, I want to create my low priority container with revocable resources only.
 5. As a Swarm user, I want to create my low priority container with revocable resources first, if there are no enough revocable resources, it is OK to create the container with regular resources.
 6. As a Swarm user, I want to create container with any type of resources, i.e., I do not care about the resource type, either revocable or regular resources are OK.
 7. As a Swarm user, I want to know the type of resources my containers consume.

## Design proposal

Overall, we will introduce a new cluster option / environment variable to enable Swarm to receive revocable resources from Mesos, and set a special label to Docker engine to mark the type of its resources, and when creating container, user can specify the type of the resources to create container with a constraint.

### Enable Swarm to receive revocable resources from Mesos

Introduce a new cluster option and a new environment variable for Swarm on Mesos:

 - **`mesos.enablerevocable`**: This is a cluster option of bool type.
     - **`True`**: We will set `FrameworkInfo_Capability_REVOCABLE_RESOURCES` in the `FrameworkInfo` when registering Swarm to Mesos as a framework, and Swarm will be able to receive revocable resources from Mesos.
     - **`False`** or this cluster option is not supplied: We will not set `FrameworkInfo_Capability_REVOCABLE_RESOURCES` in the `FrameworkInfo` when registering Swarm to Mesos as a framework, so Swarm will still only receive regular resources from Mesos.
 - **`SWARM_MESOS_ENABLE_REVOCABLE`**: This is an environment variable of bool type, it has exactly the same meaning with the cluster option `mesos.enablerevocable`.

If both of the cluster option and environment variable are set, the cluster option will overwrite the environment variable.

### Mark Docker engine's resource type with a special label

Introduce a special label to mark the resources type of a Docker engine, the key of this label is `res-type` and its value can be one of:

 - `regular`
 - `revocable`
 - `any`

When Swarm receives an offer from Mesos for a Mesos agent for the first time, 

 - If the offer only includes regular resources, add a label `res-type=regular` to the related engine (`cluster.Engine`).
 - If the offer only includes revocable resources, add a label `res-type=revocable` to the related engine.
 - If the offer includes both regular and revocable resources, add a label `res-type=any` to the related engine.

Later on, when Swarm receives more offers for the Mesos agent, it will change the label's value based on its current value and the resource type of the new offer.

 - if the label's current value is `regular`, and a new offer which includes revocable resources is received, Swarm will change its value to `any`, and vice verse.
 - If the label's current value is `any`, it will not be changed regardless the resource type of the new offers.

If an offer is removed for a Mesos agent, Swarm will also update the label accordingly.

 - If all the resources of the Mesos agent are regular after the offer is removed, Swarm will update the label's value to `regular`.
 - If all the resources of the Mesos agent are revocable after the offer is removed, Swarm will update the label's value to `revocable`.

### Use revocable and regular resource by constraint

When user runs "`docker run ...`" to create a container via Swarm, s/he can specify the type of resources to be used for the container with a constraint. 

 - **`constraint:res-type==regular`**: This is to support use case 2. With this constraint specified, when Swarm wraps engines as a candidate node list for scheduler, 
     - For the engine which has a label `res-type=any`, we will set a label `res-type=regular` for the wrapped node, and set the node's total resources to the engine's regular resources because user does not want to create the container with revocable resources.
     - For the engine which has a label `res-type=regular`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=regular`.
     - For the engine which has a label `res-type=revocable`, do not wrap them as candidate nodes since user does not want to create the container with revocable resources.
     - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If scheduler fails to return any nodes (i.e., no node has enough regular resources), return an error.
 - **`constraint:res-type==~regular`**: This is to support use case 3, we will handle it with two phases node selection:
     - Phase 1: Try to find a node which has enough regular resources: 
         - For the engine which has a label `res-type=any`, we will set a label `res-type=regular` for the wrapped node, and set the node's total resources to the engine's regular resources.
         - For the engine which has a label `res-type=regular`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=regular`.
         - For the engine which has a label `res-type=revocable`, do not wrap them as candidate nodes.
         - Temporarily change the constraint to `constraint:res-type==regular`, and then call scheduler.
         - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If scheduler fails to return any nodes (i.e., no node has enough regular resources), go to phase 2.
     - Phase 2: Try to find a node which has enough revocable resources: 
         - For the engine which has a label `res-type=any`, we will set a label `res-type=revocable` for the wrapped node, and set the node's total resources to the engine's revocable resources.
         - For the engine which has a label `res-type=revocable`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=revocable`.
         - For the engine which has a label `res-type=regular`, do not wrap them as candidate nodes.
         - Temporarily change the constraint to `constraint:res-type==revocable`, and then call scheduler.
         - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If schedulers fails to return any nodes (i.e., no node has enough revocable resources), return an error.
     - So basically we handle this constraint in the way of handling a `constraint:res-type==regular` followed by a `constraint:res-type==revocable`.
 - **`constraint:res-type==revocable`**: This is to support use case 4. With this constraint specified, when Swarm wraps engines as a candidate node list for scheduler, 
    - For the engine which has a label `res-type=any`, we will set a label `res-type=revocable` for the wrapped node, and set the node's total resources to the engine's revocable resources because user does not want to create the container with regular resources.
     - For the engine which has a label `res-type=revocable`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=revocable`.
     - For the engine which has a label `res-type=regular`, do not wrap them as candidate nodes since user does not want to create the container with regular resources.
     - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If scheduler fails to return any nodes (i.e., no node has enough revocable resources), return an error.
 - **`constraint:res-type==~revocable`**: This is to support use case 5, we will handle it with two phases node selection:
     - Phase 1: Try to find a node which has enough revocable resources: 
         - For the engine which has a label `res-type=any`, we will set a label `res-type=revocable` for the wrapped node, and set the node's total resources to the engine's revocable resources.
         - For the engine which has a label `res-type=revocable`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=revocable`.
         - For the engine which has a label `res-type=regular`, do not wrap them as candidate nodes.
         - Temporarily change the constraint to `constraint:res-type==revocable`, and then call scheduler.
         - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If scheduler fails to return any nodes (i.e., no node has enough revocable resources), go to phase 2.
     - Phase 2: Try to find a node which has enough regular resources: 
         - For the engine which has a label `res-type=any`, we will set a label `res-type=regular` for the wrapped node, and set the node's total resources to the engine's regular resources.
         - For the engine which has a label `res-type=regular`, we just follow the existing logic to wrap them as nodes, no changes needed, i.e., the wrapped node also has a label `res-type=regular`.
         - For the engine which has a label `res-type=revocable`, do not wrap them as candidate nodes.
         - Temporarily change the constraint to `constraint:res-type==regular`, and then call scheduler.
         - After scheduler returns a list of nodes by applying filters and strategy, we will use the first node in the list to create the container. If schedulers fails to return any nodes (i.e., no node has enough regular resources), return an error.
     - So basically we handle this constraint in the way of handling a `constraint:res-type==revocable` followed by a `constraint:res-type==regular`.
 - **`constraint:res-type==*`**: This is to support use case 6, we just follow the existing logic to wrap all engines as a candidate node list for scheduler. And after scheduler returns a list of nodes by applying filters and strategy, we will create the container with the first node which has enough resources (try regular resources first and then revocable resources). If schedulers fails to return any nodes, or there is no node has enough regular or revocable resources in the node list returned by scheduler, return an error.

Besides the above, user can actually specify the constrain with more forms (see below for a few examples), we will internally map them to one of the above forms.

 - `constraint:res-type!=revocable`: Mapped to `constraint:res-type==regular`.
 - `constraint:res-type==revoca*`: Mapped to `constraint:res-type==revocable`.
 - `constraint:res-type==re*`: Mapped to `constraint:res-type==*`.

> **Note:** 
> `constraint:res-type!=re*` is interpreted as user does not want to create container with any type of resources, in this case, we will return an error. And when creating a container, user can specify only one constraint with `res-type` as its key, if s/he specifies multiple, e.g., specify both `constraint:res-type==regular` and `constraint:res-type==~revocable`, we will return an error.

An alternative is, for the constraint with `res-type` as key, we only support `==` as operator, and only support one of `regular`, `~regular`, `revocable` and `~revocable` as its value, if the constraint specified by user is in any other forms, we just return an error. This may simplify our implementation, but user loses some flexibilities.

If user does not specify a constraint with `res-type` as key when creating a container, we will handle it in the same way of handling `constraint:res-type==regular` (i.e., create the container with regular resources only), this is to keep the backward compatibility.

### Mark container's resource type with a special label

After a node is selected with steps described in the last section to create the container, internally we add a label to the container to mark the actual type of resources it uses.

 - **`res-type=<type of resources>`**: Here the value can be either `regular` or `revocable` which depends on the actual type of the resources used to create the container.

The reason we want to add this label to the container is, once the container is created, when user queries the info of the container, s/he can clearly knows the type of the resources the container consumes, and such info can be used to implement some higher level business logics, e.g., billing / chargeback (e.g., the prices of regular memory and revocable memory are different).
