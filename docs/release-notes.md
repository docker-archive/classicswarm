<!--[metadata]>
+++
title = "Docker Swarm Release Notes"
description = "Docker Swarm release notes"
keywords = ["docker, swarm, clustering, discovery, release,  notes"]
[menu.main]
parent = "mn_about"	
weight = 1
+++
<![end-metadata]-->

# Docker Swarm release notes
(June 16, 2015)

For complete information on this release, see the [0.3.0 Milestone project
page](https://github.com/docker/swarm/wiki/0.3.0-Milestone-Project-Page). In
addition to bug fixes and refinements, this release adds the following:

- **API Compatibility**: Swarm is now 100% compatible with the Docker API. Everything that runs on top of the Docker API should work seamlessly with a Docker Swarm.

- **Mesos**: Experimental support for Mesos as a cluster driver.

- **Multi-tenancy**: Multiple swarm replicas of can now run in the same cluster and will automatically fail-over (using leader election).

This is our most stable release yet. We built an exhaustive testing infrastructure (integration, regression, stress).We've almost doubled our test code coverage which resulted in many bug fixes and stability improvements.

Finally, we are garnering a growing community of Docker Swarm contributors. Please consider contributing by filling bugs, feature requests, commenting on existing issues, and, of course, code.



