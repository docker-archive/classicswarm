# Changelog

## 1.1.1 (2016-02-17)

- Performance improvements around networking with Docker engine 1.10
- Fix reschedule issue regarding events
- Implement engine refresh backoff strategy for failing nodes

## 1.1.0 (2016-02-04)

#### Scheduler

- Add support for container rescheduling on node failure. (experimental)
- Use failureCount as a secondary health indicator
- Add swarm container create retry option
- Fixed the way soft affinities and handled

#### API

- Support private registry on docker run
- Expose error and last update time in docker info
- Sort images by Created
- Fix error when inspect on unhealthy node
- Prevent panic in filters when container has no name
- Add buildtime, kernelversion and experimental to API version
- Support docker update and new networking related flags in run & network create/connect/disconnect/ls
- Require `--all` on docker ps to display containers on unhealthy nodes
- Retry on docker events EOF

#### Node Management

- Add a random delay to avoid synchronized registration at swarm join
- Use engine connection error to fail engine fast
- Introduce pending state

#### Mesos integration

- Rename slave to agent
- Upgrade tests to use mesos 0.25
- Code refactors
- Improve debug output
- Enable checkpoint failover in FrameworkInfo
- Fix timeout when pulling images
- Add timeout to refuse offers
- Fix double start issue

#### Misc

- Fix license grant
- Documentation update
- Use discovery from docker/docker

## 1.0.1 (2015-12-09)

#### Scheduler

- Set labels for pending containers to fix scheduler failure

#### Discovery

- Increase default TTL and heartbeat values to reduce traffic to discovery

#### API

- Fix 'ps -a' panic issue
- Fix network connect/disconnect for overlay network
- Fix connection leak on TLS connections
- Fix CLI hang on events command
- Fix newline issue with events
- Improve OPTIONS handler
- Fix image digest
- Fix memoryswappiness default value
- Enable profiling for HTTP in debug mode

#### Node Update

- Provide options on swarm node update frequency

#### Mesos integration

- Change offers timeout default to prevent other frameworks starvation
- Improve error output for bad swarm mesos user
- Fix connection failure when using Mesos with ZooKeeper

#### Misc

- Update to Go 1.5.2
- Documentation update

## 1.0 (2015-10-13)

#### Scheduler

- Swarm is now pulling Images in parallel after the scheduling decision has been made, which mitigates the error happening occasionally with Docker pulls blocking the entire scheduler.

#### General stability

- The Node refresh loop process has been improved and does not yield to a panic when removing an Engine and trying to refresh the state at the same time.
- The refresh loop has been randomized to better handle huge scale scenarios (> 1000 nodes) where a *refresh burst* could occur and make the Manager unstable/fail.
- General improvements and fixes for the Mesos experimental backend for swarm.

#### Integration with libnetwork / Support for overlay networking

- It is now possible to use the new networking features with Swarm through the `network` sub-system. Commands like `docker network create`, `docker network attach` and `docker network ls` are now available through swarm.

#### Integration with Docker Volume Plugins

- You can now use the docker volume plugin subsystem available with `docker volume`.

#### Leader Election

- You can now specify the `--replication-ttl` flag to control how long it takes for Replicas to be notified of the Primary failure and take over the lead.

#### TLS

- This is now possible to use TLS with discovery for `consul` and `etcd` using the new `--discovery-opt` flag.

## 0.4 (2015-08-04)

#### Scheduler

- Reschedule with soft affinity when image cannot be pulled

#### Store

- Replace the store pkg by libKV
- Fixes about consul/etcd and zookeeper

#### API

- Fix docker push name matching
- Fix docker exec status code
- Fix docker pull status code

#### Docker Engine Compatibility

- Improve docker info with docker client 1.7.x
- Add SystemTime, http_proxy, https_proxy and no_proxy to docker info

#### Mesos integration

- Task creation timeout configurable
- Fix issue with hostname in library image
- Use 'docker_port' attribute if available
- Add support for random ports

#### Misc

- Add doc on leader election / high availability of swarm manager
- Lots of typos/improvements to the doc
- Switch to golang 1.4
