<!--[metadata]>
+++
title = "Docker Swarm recheduling"
description = "Swarm rescheduling"
keywords = ["docker, swarm, clustering, rescheduling"]
[menu.main]
parent="smn_workw_swarm"
weight=5
+++
<![end-metadata]-->

# Rescheduling

The Docker Swarm scheduler is able to detect node failure and
restart its containers on another node.

## Rescheduling policies

The rescheduling policies are:

* `on-node-failure`
* `off` (default if not specified)


When you start a container, use the env var `reschedule` or the
label `com.docker.swarm.reschedule-policy` to specify the policy to
apply to the container.

```
# do not reschedule (default)
$ docker run -d -e reschedule:off redis
# or
$ docker run -d -l 'com.docker.swarm.reschedule-policy=["off"]' redis
```

```
# reschedule on node failure
$ docker run -d -e reschedule:on-node-failure redis
# or
$ docker run -d -l 'com.docker.swarm.reschedule-policy=["on-node-failure"]' redis
```