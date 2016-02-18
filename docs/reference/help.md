<!--[metadata]>
+++
title = "help"
description = "Get help for Swarm commands."
keywords = ["swarm, help"]
[menu.main]
identifier="swarm.help"
parent="smn_swarm_subcmds"
+++
<![end-metadata]-->

# help - Display information about a command

The `help` command displays information about how to use a command.

For example, to see a list of Swarm options and commands, enter:

    $ docker run swarm --help

To see a list of arguments and options for a specific Swarm command, enter:

    $ docker run swarm <command> --help

For example:

    $ docker run swarm list --help
    Usage: swarm list [OPTIONS] <discovery>

    List nodes in a cluster

    Arguments:
       <discovery>    discovery service to use [$SWARM_DISCOVERY]
                       * token://<token>
                       * consul://<ip>/<path>
                       * etcd://<ip1>,<ip2>/<path>
                       * file://path/to/file
                       * zk://<ip1>,<ip2>/<path>
                       * [nodes://]<ip1>,<ip2>

    Options:
       --timeout "10s"							timeout period
       --discovery-opt [--discovery-opt option --discovery-opt option]	discovery options
