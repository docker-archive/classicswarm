<!--[metadata]>
+++
title = "Docker Swarm"
description = "Docker Swarm release notes"
keywords = ["docker, swarm, clustering, discovery, release,  notes"]
[menu.main]
parent = "smn_release_notes"	
weight = 5
+++
<![end-metadata]-->

# Docker Swarm

This page shows the cumulative release notes across all releases of Docker Swarm.

## Version 0.3.0 (June 16, 2015)

For complete information on this release, see the
[0.3.0 Milestone project page](https://github.com/docker/swarm/wiki/0.3.0-Milestone-Project-Page).
In addition to bug fixes and refinements, this release adds the following:

- **API Compatibility**: Swarm is now 100% compatible with the Docker API. Everything that runs on top of the Docker API should work seamlessly with a Docker Swarm.

- **Mesos**: Experimental support for Mesos as a cluster driver.

- **Multi-tenancy**: Multiple swarm replicas of can now run in the same cluster and will automatically fail-over (using leader election).

This is our most stable release yet. We built an exhaustive testing infrastructure (integration, regression, stress).We've almost doubled our test code coverage which resulted in many bug fixes and stability improvements.

Finally, we are garnering a growing community of Docker Swarm contributors. Please consider contributing by filling bugs, feature requests, commenting on existing issues, and, of course, code.


## Version 0.2.0 (April 16, 2015)

For complete information on this release, see the
[0.2.0 Milestone project page](https://github.com/docker/swarm/wiki/0.2.0-Milestone-Project-Page).
In addition to bug fixes and refinements, this release adds the following:

* **Increased API coverage**: Swarm now supports more of the Docker API. For
details, see
[the API README](https://github.com/docker/swarm/blob/master/api/README.md).

* **Swarm Scheduler**: Spread strategy. Swarm has a new default strategy for
ranking nodes, called "spread". With this strategy, Swarm will optimize
for the node with the fewest running containers. For details, see the
[scheduler strategy README](https://github.com/docker/swarm/blob/master/scheduler/strategy/README.md)

* **Swarm Scheduler**: Soft affinities. Soft affinities allow Swarm more flexibility
in applying rules and filters for node selection. If the scheduler can't obey a
filtering rule, it will discard the filter and fall back to the assigned
strategy. For details, see the [scheduler filter README](https://github.com/docker/swarm/tree/master/scheduler/filter#soft-affinitiesconstraints).

* **Better integration with Compose**: Swarm will now schedule inter-dependent
containers on the same host. For details, see
[PR #972](https://github.com/docker/compose/pull/972).
