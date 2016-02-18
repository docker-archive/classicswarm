<!--[metadata]>
+++
title = "create"
description = "Create a Swarm manager."
keywords = ["swarm, create"]
[menu.main]
identifier="swarm.create"
parent="smn_swarm_subcmds"
+++
<![end-metadata]-->

# create  — Create a discovery token

The `create` command uses Docker Hub's hosted discovery backend to create a unique *discovery token* for your cluster. For example:

    $ docker run --rm  swarm create
    86222732d62b6868d441d430aee4f055

Later, when you use [`manage`](manage.md) or [`join`](join.md) to create Swarm managers and nodes, you use the discovery token in the `<discovery>` argument (e.g., `token://86222732d62b6868d441d430aee4f055`). The discovery backend registers each new Swarm manager and node that uses the token as a member of your cluster.

Some documentation also refers to the discovery token as a *cluster_id*.

> Warning: Docker Hub's hosted discovery backend is not recommended for production use. It’s intended only for testing/development.

For more information and examples about this and other discovery backends, see the [Docker Swarm Discovery](../discovery.md) topic.
