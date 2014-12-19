Docker Swarm Roadmap
=====================

####Security
* [x] TLS

####Scheduler
* [ ] storage + Virtual Container ID
* [ ] rebalancing
* [ ] affinity constraints + improved constraints expression (==, !=)
* [ ] global scheduling

####Multi-tenancy
* [ ] master election
* [ ] shared state

####API Matching
* [ ] docker attach

####Extensibility
* [ ] pluggable scheduler
* [ ] discovery backends
  * [x]    etcd
  * [ ]    zookeeper
  * [x]    hub 
  * [x]    file
