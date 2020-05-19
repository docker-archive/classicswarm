---
advisory: swarm-standalone
hide_from_sitemap: true
description: Swarm rescheduling
keywords: docker, swarm, clustering, rescheduling
title: Swarm rescheduling
---

You can set rescheduling policies with Docker Swarm. A rescheduling policy
determines what the Swarm scheduler does for containers when the nodes they are
running on fail.

## Rescheduling policies

You set the reschedule policy when you start a container. You can do this with
the `reschedule` environment variable or the
`com.docker.swarm.reschedule-policies` label. If you don't specify a policy, the
default rescheduling policy is `off` which means that Swarm does not restart a
container when a node fails.

To set the `on-node-failure` policy with a `reschedule` environment variable:

```bash
$ docker run -d -e "reschedule:on-node-failure" redis
```

To set the same policy with a `com.docker.swarm.reschedule-policies` label:

```bash
$ docker run -d -l 'com.docker.swarm.reschedule-policies=["on-node-failure"]' redis
```

## Review reschedule logs

You can use the `docker logs` command to review the rescheduled container
actions. To do this, use the following command syntax:

```bash
docker logs SWARM_MANAGER_CONTAINER_ID
```

When a container is successfully rescheduled, it generates a message similar to
the following:

```none
Rescheduled container 2536adb23 from node-1 to node-2 as 2362901cb213da321
Container 2536adb23 was running, starting container 2362901cb213da321
```

If for some reason, the new container fails to start on the new node, the log
contains:

```none
Failed to start rescheduled container 2362901cb213da321
```

## Related information

* [Apply custom metadata](/engine/userguide/labels-custom-metadata/)
* [Environment variables with run](/engine/reference/run/#env-environment-variables)
