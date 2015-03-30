Docker Swarm Roadmap
=====================

####Scheduler
Swarm comes with a builtin scheduler. It currently provides basic functionalities, such as
scheduling containers based on constraints or affinity (co-scheduling of containers), persistent
storage and multiple scheduling strategies like binpacking or random.

We plan to add more features to the builtin scheduler such as rebalancing (in case of host failure)
and global scheduling (schedule containers on every node)

* [ ] Virtual Container ID
* [ ] Rebalancing
* [ ] Global scheduling

####Leader Election (Distributed State)
Regarding Swarm Multi-tenancy, we are working on shared states between multiple "soon-to-be-master"
and master election.

* [ ] Shared state
* [ ] Master election

####API Matching
Swarm currently supports around 75% of the Docker API as you can see [here](https://github.com/docker/swarm/blob/master/api/README.md)

Our goal is to support 100% of the API, so all the Docker CLI commands would work against Swarm 

* [ ] Bring Swarm API on par with Docker API as much as possible

####Extensibility
The builtin scheduler allows you to manage container on approximately hundreds of machines within a cluster.
In the future, we want to allow third party tools to be plugged into Docker Swarm to allow management
of much bigger clusters.

The first integration will be Mesos but other tools will arrive after, like kubernetes.

* [x] Cluster Drivers [#393](https://github.com/docker/swarm/issues/393)
* [ ] Mesos implementation

Project Planning
================

An [Open-Source Planning Process](https://github.com/docker/swarm/wiki/Open-Source-Planning-Process) is used to define the Roadmap. [Project Pages](https://github.com/docker/swarm/wiki) define the goals for each Milestone and identify current progress.

| Platform Version | Swarm Milestone | Project Page | 
|------------------|-----------------|--------------|
| Docker 1.6.0 | [0.2.0](https://github.com/docker/swarm/milestones/0.2.0) | [Project Page](https://github.com/docker/swarm/wiki/0.2.0-Milestone-Project-Page) | 
| Docker 1.7.0 | [1.0.0](https://github.com/docker/swarm/milestones/1.0.0) | [Project Page](https://github.com/docker/swarm/wiki/1.0.0-Milestone-Project-Page) | 
