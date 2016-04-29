<!--[metadata]>
+++
title = "swarm"
description = "swarm command overview"
keywords = ["Swarm, cluster, commands"]
[menu.main]
identifier="swarm.swarm"
parent="smn_swarm_subcmds"
+++
<![end-metadata]-->

# Swarm — A Docker-native clustering system

The `swarm` command runs a Swarm container on a Docker Engine host and performs the task specified by the required subcommand, `COMMAND`.

Use `swarm` with the following syntax:

    $ docker run swarm [OPTIONS] COMMAND [arg...]

For example, you use `swarm` with the `manage` subcommand to create a Swarm manager in a high-availability cluster with other managers:

    $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000 consul://172.30.0.165:8500

## Options

The `swarm` command has the following options:

* `--debug` — Enable debug mode. Display messages that you can use to debug a Swarm node. For example:
        time="2016-02-17T17:57:40Z" level=fatal msg="discovery required to join a cluster. See 'swarm join --help'."
  The environment variable for this option is `[$DEBUG]`.
* `--log-level "<value>"` or `-l "<value>"` — Set the log level. Where `<value>` is: `debug`, `info`, `warn`, `error`, `fatal`, or `panic`. The default value is `info`.
* `--experimental` — Enable experimental features.
* `--help` or `-h` — Display help.
*  `--version` or `-v` — Display the version. For example:
        $ docker run swarm --version
        swarm version 1.1.0 (a0fd82b)

## Commands

The `swarm` command has the following subcommands:

- [create, c](create.md) - Create a discovery token
- [list, l](list.md) - List the nodes in a Docker cluster
- [manage, m](manage.md) - Create a Swarm manager
- [join, j](join.md) - Create a Swarm node
- [help](help.md) - Display a list of Swarm commands, or help for one command
