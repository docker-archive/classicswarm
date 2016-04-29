# Swarm: a Docker-native clustering system

[![GoDoc](https://godoc.org/github.com/docker/swarm?status.png)](https://godoc.org/github.com/docker/swarm)
[![Jenkins Build Status](https://jenkins.dockerproject.org/view/Swarm/job/Swarm%20Master/badge/icon)](https://jenkins.dockerproject.org/view/Swarm/job/Swarm%20Master/)
[![Build Status](https://travis-ci.org/docker/swarm.svg?branch=master)](https://travis-ci.org/docker/swarm)
[![Go Report Card](https://goreportcard.com/badge/github.com/docker/swarm)](https://goreportcard.com/report/github.com/docker/swarm)

![Docker Swarm Logo](logo.png?raw=true "Docker Swarm Logo")

Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
into a single, virtual host.

Swarm serves the standard Docker API, so any tool which already communicates
with a Docker daemon can use Swarm to transparently scale to multiple hosts:
Dokku, Compose, Krane, Flynn, Deis, DockerUI, Shipyard, Drone, Jenkins... and,
of course, the Docker client itself.

Like other Docker projects, Swarm follows the "batteries included but removable"
principle. It ships with a set of simple scheduling backends out of the box, and as
initial development settles, an API will be developed to enable pluggable backends.
The goal is to provide a smooth out-of-the-box experience for simple use cases, and
allow swapping in more powerful backends, like Mesos, for large scale production
deployments.

## Installation for Swarm Users

For instructions on using Swarm in your dev, test or production environment, refer to the Docker Swarm documentation on [docs.docker.com](http://docs.docker.com/swarm/).

## Building Swarm from Source

To compile Swarm from source code, refer to the instructions in [CONTRIBUTING.md](http://github.com/docker/swarm/blob/master/CONTRIBUTING.md)

## Participating

You can contribute to Docker Swarm in several different ways:

  - If you have comments, questions, or want to use your knowledge to help others, come join the conversation on IRC. You can reach us at #docker-swarm on Freenode.

  - To report a problem or request a feature, please file an issue.

  - Of course, we welcome pull requests and patches.  Setting up a local Swarm development environment and submitting PRs is described [here](http://github.com/docker/swarm/blob/master/CONTRIBUTING.md).

Finally, if you want to see what we have for the future and learn more about our release cycles, all this information is detailed on the [wiki](https://github.com/docker/swarm/wiki)

## Copyright and license

Copyright © 2014-2016 Docker, Inc. All rights reserved, except as follows. Code is released under the Apache 2.0 license. The README.md file, and files in the "docs" folder are licensed under the Creative Commons Attribution 4.0 International License under the terms and conditions set forth in the file "LICENSE.docs". You may obtain a duplicate copy of the same license, titled CC-BY-SA-4.0, at http://creativecommons.org/licenses/by/4.0/.
