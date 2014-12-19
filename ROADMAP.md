Docker Swarm Roadmap
=====================

####Security
* [x] TLS authentication

####Scheduler
* [ ] Persistent state storage
* [ ] Virtual Container ID
* [ ] Rebalancing
* [ ] Affinity constraints & improved constraints expression (==, !=, regular expressions)
* [ ] Global scheduling (schedule containers on every node)

####Multi-tenancy
* [ ] Master election
* [ ] Shared state

####API Matching
* [ ] Bring Swarm API on par with Docker API as much as possible
* [ ] Support for `docker attach` (interactive sessions)

####Extensibility
* [ ] Pluggable scheduler
* [ ] Discovery backends
  * [x]    etcd
  * [ ]    zookeeper
  * [x]    hub 
  * [x]    file
