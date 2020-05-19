---
title: Docker Swarm (standalone) release notes
description: Release notes for Docker Swarm (standalone)
keywords: release notes, swarm
toc_max: 2
redirect_from:
  - /release-notes/docker-swarm/
---

## 1.2.6 (2017-01-17)

### API

- Add options to refresh engine container cache on filters
- Support name filter in volume list
- Support more filters in network list operations
- Support node whitelist filter
- Switch from engine-api to docker/api/type and docker/clients
- Pass size parameter through on container inspect

### Scheduler

- For container network disconnect requests, tryworker engine with the container first
- Precompile filter regular expression to reduce PU usage
- Remove setTCPUserTimeout to avoid TCP connection leak
- Fix network endpoints for rescheduling
- Give up leadership when manager shuts down

### Cluster management

- Remove dependency on IPv4 addresses
- Support event top, resize, commit and so on to avoid unnecessary refreshing
- Sequentialize event monitor to an engine to avoid data race
- When an active engine sends EOF on event stream, restart event monitor so we don't lose events
- When proxying a request, cancel request if user connection is broken

### MISC

- Update go-zookeeper to fix a lock spin problem
- Migrate documentation to https://github.com/docker/docker.github.io/tree/master/swarm
- Update Swarm CI to use go 1.7.1
- Support GOARCH to be able to build for other architectures
- Send Swarm logs to stdout

## 1.2.5 (2016-08-18)

### Scheduler

- Fix container rescheduling with overlay network
- Fix scheduler detail log improper effect when container name is empty
- Check unique container name on create and rename for Mesos cluster

### Health check

- Refresh container status on health_status events

### Doc

- Fix install-w-machine.md using docker-machine --swarm feature

## 1.2.4 (2016-07-28)

### API

- New client interface in Swarm, to differentiate from Swarm mode in Docker 1.12
- Underlying HTTP client for API is created inside Swarm
- Update minimum Docker Engine version supported by Swarm to 1.8
- Additional error handling
- Code refactoring

### Networking

- Fix concurrent map writes race condition
- Refresh single network when network event is emitted (performance improvement)
- Avoid network refresh when creating container (performance improvement)

### Volumes

- Refresh single volume when volume event is emitted (performance improvement)
- Avoid volume refresh when creating container (performance improvement)

### Events

- Support daemon events for Swarm

### Test

- Fix leader election tests
- Fix rescheduling test

### Mesos

- Fix double locking issue

### Misc

- Handle systime difference between Swarm and Engines
- Add healthcheck information to CLI
- Fix `engine_reconnect` issue that led to reconnected engine being treated as new

## 1.2.3 (2016-05-25)

### API

- Update `engine-api` vendoring (supports new functions and signatures)
- Fix registry auth bug for image pulls

## 1.2.2 (2016-05-05)

### Cluster management

- Fix deadlock that causes Swarm to hang

## 1.2.1 (2016-05-03)

### Scheduler

- Add containerslots filter to allow user to limit container number on a node

### API

- Use engine-api to handle large number of API calls
- Update ContainerConfig to embed HostConfig and NetworkingConfig
- stop/restart/kill a non-existent container should return 500 rather than 404
- Return an error when assertion fails in hijack
- Return an error when Image Pull fails
- Fix image pull bug (wait until download finishes)
- Fix and document some api response status codes
- Add NodeID in docker info
- Support docker ps --filter by volume

### Build

- Move dependencies to vendor/
- Update Image Pull to use docker/distribution package
- Convert docs Dockerfiles to use docs/base:oss

### Test

- Fix api/ps tests
- Update api/stats test to prevent timeout on master branch
- Use --cpu-shares instead of -c in integration test

### Misc

- Documentation clean up
- Close http response body to avoid potential memory leak
- Switch context.TODO() to context.Background() to enable context setting

## 1.2.0 (2016-04-13)

### Scheduler

- Move rescheduling out of experimental
- Differentiate constraint errors from affinity errors
- Printing unsatisfiable constraints for container scheduling failure
- Enable rescheduling on master manager to prevent replica managers from rescheduling containers
- Output error when starting a rescheduled container fails, and when removing container fails at node recovery
- Validate cluster swarm.overcommit option

### API

- Introduce engine-api client to Swarm
- Implement 'info' and 'version' with engine-api
- Use apiClient for some volume, network, image operations
- Print engine version in Info
- Fix swarm api response status code
- Support ps node filter
- Fix HostConfig for /start endpoint
- Print container 'created' state at ps
- Update dockerclient to get labels on volumes, networks, images
- Support private images, labels and other new flags in docker build
- Select apiClient version according to node docker version

### Node management

- Prevent crash on channel double close
- Manager retries EventMonitoring on failure.
- Docker engine updates hostname/domainname
- Force inspect for containers in Restarting state.
- Increase max thread count to 50k to accommodate large cluster or heavy workload
- Force to validate min and max refresh interval to be positive
- Skip unstable tests from Docker bug 14203
- Fix race condition between node removal from discovery and scheduler
- Fix data race with node failureCount
- Display warning message if an engine has labels with "node=xxx"

### Discovery

- Remove parameter which is not used in createDiscovery
- Fix Consul leader election failure on multi-server

### Mesos integration

- Support rescind offer in swarm
- Update mesos tests

### Misc

- Update golang version to 1.5.4
- Skip redundant endpoints in "network inspect"
- Validate duration flags:--delay, --timeout, --replication-ttl
- Fix image matching via id
- Make port 0 invalid as listening port
- Improve volume inspect test
- Add read lock for eventsHandler when only it is necessary

## 1.1.3 (2016-03-03)

- Fix missing HostConfig for rescheduled containers
- Fix TCP connections leak
- Support `docker run --net <node>/<network> ...`
- Fix CORS issue in the API

## 1.1.2 (2016-02-18)

- Fix regression with Docker Compose

## 1.1.1 (2016-02-17)

- Performance improvements around networking with Docker engine 1.10
- Fix reschedule issue regarding events
- Implement engine refresh backoff strategy for failing nodes

## 1.1.0 (2016-02-04)

### Scheduler

- Add support for container rescheduling on node failure. (experimental)
- Use failureCount as a secondary health indicator
- Add swarm container create retry option
- Fix the way soft affinities and handled

### API

- Support private registry on docker run
- Expose error and last update time in docker info
- Sort images by Created
- Fix error when inspect on unhealthy node
- Prevent panic in filters when container has no name
- Add buildtime, kernelversion and experimental to API version
- Support docker update and new networking related flags in run & network create/connect/disconnect/ls
- Require `--all` on docker ps to display containers on unhealthy nodes
- Retry on docker events EOF

### Node management

- Add a random delay to avoid synchronized registration at swarm join
- Use engine connection error to fail engine fast
- Introduce pending state

### Mesos integration

- Rename slave to agent
- Upgrade tests to use mesos 0.25
- Code refactors
- Improve debug output
- Enable checkpoint failover in FrameworkInfo
- Fix timeout when pulling images
- Add timeout to refuse offers
- Fix double start issue

### Misc

- Fix license grant
- Documentation update
- Use discovery from docker/docker

## 1.0.1 (2015-12-09)

### Scheduler

- Set labels for pending containers to fix scheduler failure

### Discovery

- Increase default TTL and heartbeat values to reduce traffic to discovery

### API

- Fix 'ps -a' panic issue
- Fix network connect/disconnect for overlay network
- Fix connection leak on TLS connections
- Fix CLI hang on events command
- Fix newline issue with events
- Improve OPTIONS handler
- Fix image digest
- Fix memoryswappiness default value
- Enable profiling for HTTP in debug mode

### Node update

- Provide options on swarm node update frequency

### Mesos integration

- Change offers timeout default to prevent other frameworks starvation
- Improve error output for bad swarm mesos user
- Fix connection failure when using Mesos with ZooKeeper

### Misc

- Update to Go 1.5.2
- Documentation update

## 1.0 (2015-10-13)

### Scheduler

- Swarm is now pulling Images in parallel after the scheduling decision has been made, which mitigates the error happening occasionally with Docker pulls blocking the entire scheduler.

### General stability

- The Node refresh loop process has been improved and does not yield to a panic when removing an Engine and trying to refresh the state at the same time.
- The refresh loop has been randomized to better handle huge scale scenarios (> 1000 nodes) where a *refresh burst* could occur and make the Manager unstable/fail.
- General improvements and fixes for the Mesos experimental backend for swarm.

### Integration with libnetwork / Support for overlay networking

- It is now possible to use the new networking features with Swarm through the `network` sub-system. Commands like `docker network create`, `docker network attach` and `docker network ls` are now available through swarm.

### Integration with Docker Volume Plugins

- You can now use the docker volume plugin subsystem available with `docker volume`.

### Leader election

- You can now specify the `--replication-ttl` flag to control how long it takes for Replicas to be notified of the Primary failure and take over the lead.

### TLS

- This is now possible to use TLS with discovery for `consul` and `etcd` using the new `--discovery-opt` flag.

## 0.4 (2015-08-04)

### Scheduler

- Reschedule with soft affinity when image cannot be pulled

### Store

- Replace the store pkg by libKV
- Fixes about consul/etcd and zookeeper

### API

- Fix docker push name matching
- Fix docker exec status code
- Fix docker pull status code

### Docker Engine compatibility

- Improve docker info with docker client 1.7.x
- Add SystemTime, http_proxy, https_proxy, and no_proxy to docker info

### Mesos integration

- Task creation timeout configurable
- Fix issue with hostname in library image
- Use 'docker_port' attribute if available
- Add support for random ports

### Misc

- Add doc on leader election / high availability of swarm manager
- Lots of typos/improvements to the doc
- Switch to golang 1.4
