<!--[metadata]>
+++
title = "Docker Deprecated Features"
description = "Deprecated Features."
keywords = ["docker, documentation, about, technology, deprecate"]
[menu.main]
parent = "mn_use_docker"
+++
<![end-metadata]-->

# Deprecated Features

The following list of features are deprecated.

### Command line short variant options
**Deprecated In Release: v1.9**

**Target For Removal In Release: v1.11**

The following short variant options are deprecated in favor of their long
variants:

    docker run -c (--cpu-shares)
    docker build -c (--cpu-shares)
    docker create -c (--cpu-shares)

### Driver Specific Log Tags
**Deprecated In Release: v1.9**

**Target For Removal In Release: v1.11**

Log tags are now generated in a standard way across different logging drivers.
Because of which, the driver specific log tag options `syslog-tag`, `gelf-tag` and
`fluentd-tag` have been deprecated in favor of the generic `tag` option.

    docker --log-driver=syslog --log-opt tag="{{.ImageName}}/{{.Name}}/{{.ID}}"

### LXC built-in exec driver
**Deprecated In Release: v1.8**

**Target For Removal In Release: v1.10**

The built-in LXC execution driver is deprecated for an external implementation.
The lxc-conf flag and API fields will also be removed.

### Old Command Line Options
**Deprecated In Release: [v1.8.0](../release-notes.md#docker-engine-1-8-0)**

**Target For Removal In Release: v1.10**

The flags `-d` and `--daemon` are deprecated in favor of the `daemon` subcommand:

    docker daemon -H ...

The following single-dash (`-opt`) variant of certain command line options
are deprecated and replaced with double-dash options (`--opt`):

    docker attach -nostdin
    docker attach -sig-proxy
    docker build -no-cache
    docker build -rm
    docker commit -author
    docker commit -run
    docker events -since
    docker history -notrunc
    docker images -notrunc
    docker inspect -format
    docker ps -beforeId
    docker ps -notrunc
    docker ps -sinceId
    docker rm -link
    docker run -cidfile
    docker run -cpuset
    docker run -dns
    docker run -entrypoint
    docker run -expose
    docker run -link
    docker run -lxc-conf
    docker run -n
    docker run -privileged
    docker run -volumes-from
    docker search -notrunc
    docker search -stars
    docker search -t
    docker search -trusted
    docker tag -force

The following double-dash options are deprecated and have no replacement:

    docker run --networking
    docker ps --since-id
    docker ps --before-id
    docker search --trusted

### Auto-creating missing host paths for bind mounts
**Deprected in Release: v1.9**

**Target for Removal in Release: 1.11**

When creating a container with a bind-mounted volume-- `docker run -v /host/path:/container/path` --
docker was automatically creating the `/host/path` if it didn't already exist.

This auto-creation of the host path is deprecated and docker will error out if
the path does not exist.

### Interacting with V1 registries

Version 1.9 adds a flag (`--disable-legacy-registry=false`) which prevents the docker daemon from `pull`, `push`, and `login` operations against v1 registries.  Though disabled by default, this signals the intent to deprecate the v1 protocol.

### Docker Content Trust ENV passphrase variables name change
**Deprecated In Release: v1.9**

**Target For Removal In Release: v1.10**

As of 1.9, Docker Content Trust Offline key will be renamed to Root key and the Tagging key will be renamed to Repository key. Due to this renaming, we're also changing the corresponding environment variables

- DOCKER_CONTENT_TRUST_OFFLINE_PASSPHRASE will now be named DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE
- DOCKER_CONTENT_TRUST_TAGGING_PASSPHRASE will now be named DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE
