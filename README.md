# Classic Swarm: a Docker-native clustering system

[![GoDoc](https://godoc.org/github.com/docker/swarm?status.png)](https://godoc.org/github.com/docker/swarm)
[![Build Status](https://travis-ci.org/docker/swarm.svg?branch=master)](https://travis-ci.org/docker/swarm)
[![Go Report Card](https://goreportcard.com/badge/github.com/docker/swarm)](https://goreportcard.com/report/github.com/docker/swarm)

![Docker Swarm Logo](logo.png?raw=true "Docker Swarm Logo")

Docker Swarm "Classic" is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual host.

## Swarm Disambiguation

**Docker Swarm "Classic" standalone**: This project. A native clustering system for
Docker. It turns a pool of Docker hosts into a single, virtual host using an
API proxy system. See [Docker Swarm overview](https://docs.docker.com/swarm/overview/).
It was Docker's first container orchestration project that began in 2014.

**[Swarmkit](https://github.com/docker/swarmkit)**: Cluster
management and orchestration features in Docker Engine 1.12 or later. When Swarmkit
is enabled we call Docker Engine running in swarm mode. See the
feature list: [Swarm mode overview](https://docs.docker.com/engine/swarm/).
This project focuses on micro-service architecture. It supports service
reconciliation, load balancing, service discovery, built-in certificate rotation, etc.

## Copyright and license

Copyright © 2014-2018 Docker, Inc. All rights reserved, except as follows. Code
is released under the Apache 2.0 license. The README.md file, and files in the
"docs" folder are licensed under the Creative Commons Attribution 4.0
International License under the terms and conditions set forth in the file
"LICENSE.docs". You may obtain a duplicate copy of the same license, titled
CC-BY-SA-4.0, at http://creativecommons.org/licenses/by/4.0/.
