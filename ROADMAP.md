Docker Swarm Roadmap
=====================

####Security
* [x] TLS authentication

####Scheduler
* [ ] Persistent state storage
* [ ] Virtual Container ID
* [ ] Rebalancing
* [x] Affinity constraints & improved constraints expression (==, !=, regular expressions)
* [ ] Global scheduling (schedule containers on every node)

####Multi-tenancy
* [ ] Master election
* [ ] Shared state

####API Matching
* [ ] Bring Swarm API on par with Docker API as much as possible
* [x] Support for `docker attach` (interactive sessions)

####Extensibility
* [ ] Pluggable scheduler
* [ ] Discovery backends
  * [x]    etcd
  * [x]    zookeeper
  * [x]    consul
  * [x]    hub 
  * [x]    file
