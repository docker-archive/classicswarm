<!--[metadata]>
+++
title = "Docker network driver plugins"
description = "Network drive plugins."
keywords = ["Examples, Usage, plugins, docker, documentation, user guide"]
[menu.main]
parent = "mn_extend"
weight=-1
+++
<![end-metadata]-->

# Docker network driver plugins

Docker supports network driver plugins via
[LibNetwork](https://github.com/docker/libnetwork). Network driver plugins are
implemented as "remote drivers" for LibNetwork, which shares plugin
infrastructure with Docker. In effect this means that network driver plugins
are activated in the same way as other plugins, and use the same kind of
protocol.

## Using network driver plugins

The means of installing and running a network driver plugin will depend on the
particular plugin.

Once running however, network driver plugins are used just like the built-in
network drivers: by being mentioned as a driver in network-oriented Docker
commands. For example,

    docker network create -d weave mynet

Some network driver plugins are listed in [plugins](plugins.md)

The network thus created is owned by the plugin, so subsequent commands
referring to that network will also be run through the plugin such as,

    docker run --net=mynet busybox top

## Network driver plugin protocol

The network driver protocol, additional to the plugin activation call, is
documented as part of LibNetwork:
[https://github.com/docker/libnetwork/blob/master/docs/remote.md](https://github.com/docker/libnetwork/blob/master/docs/remote.md).

# Related GitHub PRs and issues

Please record your feedback in the following issue, on the usual
Google Groups, or the IRC channel #docker-network.

 - [#14083](https://github.com/docker/docker/issues/14083) Feedback on
   experimental networking features
