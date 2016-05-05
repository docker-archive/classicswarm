<!--[metadata]>
+++
title = "Try Swarm at scale"
description = "Try Swarm at scale"
keywords = ["docker, swarm, scale, voting, application,  certificates"]
[menu.main]
parent="workw_swarm"
identifier="scale_swarm"
weight=-35
+++
<![end-metadata]-->

# Try Swarm at scale

Using this example, you deploy a voting application on a Swarm cluster. This
example illustrates a typical development process. After you establish an
infrastructure, you create a Swarm cluster and deploy the application against
the cluster.

After building and manually deploying the voting application, you'll construct a
Docker Compose file. You (or others) can use the file to deploy and scale the
application further. The article also provides a troubleshooting section you can
use while developing or deploying the voting application.

The sample is written for a novice network administrator. You should have basic
skills on Linux systems and `ssh` experience. Some knowledge of Git is also
useful but not strictly required. This example takes approximately an hour to
complete and has the following steps:

- [Learn the application architecture](about.md)
- [Deploy your infrastructure](deploy-infra.md)
- [Deploy the application](deploy-app.md)
- [Troubleshoot the application](troubleshoot.md)
