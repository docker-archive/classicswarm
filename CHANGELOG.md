# Changelog

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
