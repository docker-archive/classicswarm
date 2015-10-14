# Changelog

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
