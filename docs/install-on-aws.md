<!--[metadata]>
+++
title = "Build a Swarm cluster for production"
description = "Deploying Swarm on AWS EC2 AMI's in a VPC"
keywords = ["docker, swarm, clustering, examples, Amazon, AWS EC2"]
[menu.main]
parent="workw_swarm"
+++
<![end-metadata]-->


# Build a Swarm cluster for production

This example shows you how to deploy a high-availability Docker Swarm cluster. Although this example uses the Amazon Web Services (AWS) platform, you can deploy an equivalent Docker Swarm cluster on many other platforms.

The Swarm cluster will consist of three types of nodes:
- Two Swarm managers
- Two Swarm nodes (aka agents)
- One discovery backend node running etcd

To start, you'll establish basic network security by creating a security group that restricts inbound traffic. Then, you'll create five hosts on your network by launching EC2 instances, applying the appropriate security group to each one, and installing Docker Engine on each one. You'll create a discovery backend by running an etcd container on one of the hosts. You'll create the Swarm cluster by running two Swarm managers in a high-availability configuration, and running two Swarm nodes. You'll communicate with the Swarm via the primary manager, running a simple hello world application and then checking which node ran the application. Lastly, you'll create an isolated virtual network for the Swarm cluster, called an overlay network.

For a gentler introduction to Swarm, try the [Evaluate Swarm in a sandbox](https://docs.docker.com/swarm/install-w-machine/) page.

<This example doesn't use Amazon's EC2 Container Service (ECS)>

## Prerequisites

- An Amazon Web Services (AWS) account
- Familiarity with AWS features and tools, such as:
  - EC2 Dashboard
  - VPC Dashboard
  - VPC Security groups
  - Connecting to an EC2 instance using SSH

## Establish basic network security

You create basic network security by restricting the types of inbound traffic that can reach the hosts on your network. To accomplish this, you create a security group and add rules to it. Each rule specifies the type, protocol, port range, and source of the traffic that is allowed to reach your hosts. This security group excludes all other inbound traffic. It also has a rule that allows all outbound traffic.

This one-size-fits-all security group is suitable for an example like this one. For a production environment, you would probably create multiple security groups with more restrictive rules for traffic to each type of host. To establish network security for your production environment, consult a network security expert.

Important: You do not need to create a VPC. New EC2 instances have a default VPC. ***When you create the following security group, associate it with the default VPC.***

From your AWS home console, click **VPC - Isolated Cloud Resources**. Then, in the VPC Dashboard that opens, navigate to **Your VPCs**.

Check the **VPC CIDR** of the default VPC (e.g., "172.30.0.0/16"). If the CIDR does not contain the "172.30.0.0" dotted quad, update the following rules with the actual dotted quad. Leave the "/24" portion of the CIDR unchanged.

Navigate to **Security Groups**. Create a security group named "Docker Swarm Example" with the following inbound rules. The **Allows** column explains what each rule allows and is just for your reference.

| Type             | Protocol   | Port Range | Source        | Allows          |
| ---------------  | -----      | -----      | -----         | -----           |
| SSH              | TCP        | 22         | 0.0.0.0/0     | SSH connection  |
| HTTP             | TCP        | 80         | 0.0.0.0/0     | Container images |
| Custom TCP Rule  | TCP        | 2375       | 172.30.0.0/24 | Non-TLS traffic |
| Custom UDP Rule  | UDP        | 4789       | 172.30.0.0/24 | Overlay network data plane (VXLAN) |
| Custom TCP Rule  | TCP        | 7946       | 172.30.0.0/24 | Overlay network control plane |
| Custom UDP Rule  | UDP        | 7946       | 172.30.0.0/24 | Overlay network control plane |
| Custom TCP Rule  | TCP        | 2379       | 172.30.0.0/24 | etcd clients    |
| Custom TCP Rule  | TCP        | 4000       | 172.30.0.0/24 | Swarm HA managers |

If you decide to create a high-availability discovery backend using three or more etcd peers, add the following rule.

| Type             | Protocol   | Port Range | Source        | Allows          |
| ---------------  | -----      | -----      | -----         | -----           |
| Custom TCP Rule  | TCP        | 2380       | 172.30.0.0/24 | etcd peers      |

## Create your hosts

Here, you create five Linux hosts that are part of the "Docker Swarm Example" security group and install Docker Engine on each one.

Open the EC2 Dashboard and launch four EC2 instances one at a time:

- During **Step 1: Choose an Amazon Machine Image (AMI)**, pick the *Amazon Linux AMI*.

- During **Step 5: Tag Instance**, under **Value**, give each instance one of these names:
    - manager0 & etcd0
    - manager1
    - node0
    - node1


- During **Step 6: Configure Security Group**, choose **Select an existing security group** and pick "Docker Swarm Example".

Review and launch your instances.

Connect to each instance using SSH and install Docker Engine [using these instructions](https://docs.docker.com/engine/installation/rhel/#install-with-the-script).

> Use the `sudo usermod -aG docker ec2-user` command mentioned by the command line output, followed by `logout`.

> Don't create an AMI image of an instance running Docker Engine for use in your Swarm cluster. Doing so causes errors

## Set up an etcd discovery backend

Here, you're going to create a minimalist discovery backend. The Swarm managers and nodes use this backend to authenticate themselves as members of the cluster. The Swarm managers also use this information to identify which nodes are available to run containers.

If you were creating a high-availability discovery backend, you would create an etcd cluster by running three etcd peers on separate dedicated hosts. However, to keep things simple for this example, you are going to run a single etcd daemon on the same host as one of the Swarm managers.

To start, copy the following launch command to a text file.

        $ docker run -d -v /usr/share/ca-certificates/:/etc/ssl/certs -p 2380:2380 -p 2379:2379 \
        --name etcd quay.io/coreos/etcd \
        -name etcd0 \
        -advertise-client-urls http://<eth0>:2379 \
        -listen-client-urls http://0.0.0.0:2379 \
        -initial-advertise-peer-urls http://<eth0>:2380 \
        -listen-peer-urls http://0.0.0.0:2380 \
        -initial-cluster-token etcd-cluster-1 \
        -initial-cluster etcd0=http://<eth0>:2380 \
        -initial-cluster-state new

Then, use SSH to connect to the "manager0 & etcd0" instance. At the command line, enter `ifconfig`. From the output, copy the IP address from `inet addr` for  `eth0`.

In the text file, find and replace <eth0> with the IP address.

Copy the launch command from the text file and paste it into the command line for on "manager0 & etc0". Your etcd node is up and running.

## Create a high-availability Swarm cluster

After creating the discovery backend, you can create the Swarm managers. Here, you are going to create two Swarm managers in a high-availability configuration. The first manager you run becomes the Swarm's *primary manager*. Some documentation refers to a primary manager as a "master", but that term has been superseded. The second manager you run serves as a *replica* instance. If the primary manager becomes unavailable, the cluster elects the replica as the primary manager.

To create the primary manager in a high-availability Swarm cluster, use the following syntax:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise <manager0_ip_address>:4000  etcd://<eth0>

Because this is particular manager is on the "manager0 & etcd0" instance, replace both `<manager0_ip_address>` and `<eth0>` with the same IP address. For example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000  etcd://172.30.0.161:2379

Enter `docker ps`. From the output, verify that both a swarm and an etcd container are running. Then, disconnect from the "manager0 & etcd0" instance.

Connect to the "manager1" instance and get its IP address. Then enter the following command, replacing `<manager1_ip_address>`. For example:

        $ docker run -d swarm manage -H :4000 --replication --advertise <manager1_ip_address>:4000  etcd://172.30.0.161:2379

Enter `docker ps`. From the output, verify that a swarm container is running.

Now, connect to each of the "node0" and "node1" instances, get the IP address,  and run a Swarm node on each one using the following syntax:

        $ docker run -d swarm join --advertise=<node_ip_address>:2375 etcd://<eth0>:2379

For example:

        $ docker run -d swarm join --advertise=172.30.0.69:2375 etcd://172.30.0.161:2379


You have finished setting up the Swarm.

## Communicate with the Swarm

You can communicate with the Swarm to get information about the managers and nodes using the Swarm API, which is nearly the same as the standard Docker API.  
In this example, you are still using the SSL connection to the "manager0 & etc0".

To get information about the master and nodes in the cluster, enter:

        $ docker -H :4000 info

The output displays information about the master and nodes, including that this master's role is primary (Role: primary).

Now run an application on the Swarm:

        $ docker -H :4000 run hello-world

To see which Swarm node ran the application, enter:

        $ docker -H :4000 ps

## Test the high-availability Swarm managers

To see the replica instance take over, you're going to shut down and restart the primary master.

Using an SSH connection to the "manager0 & etc0" instance, get the container id or name of the swarm container:

        $ docker ps

Shut down the primary master, replacing `<id_name>` with the container id or name.

        $ docker rm -f <id_name>

Restart the swarm master. For example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000  etcd://172.30.0.161:237

Look at the logs, replacing `<id_name>` with the container id or name:

       $ sudo docker logs <id_name>

The output shows will show two entries like these ones:

        time="2016-02-02T02:12:32Z" level=info msg="Leader Election: Cluster leadership lost"
        time="2016-02-02T02:12:32Z" level=info msg="New leader elected: 172.30.0.160:4000"

To get information about the master and nodes in the cluster, enter:

        docker -H :4000 info

You can connect to the "master1" node and run the `info` and `logs` commands. They will display corresponding entries for the change in leadership.

## Additional Resources

- Installing Docker Engine
    - [Example: Manual install on a cloud provider](http://docs-stage.docker.com/engine/installation/cloud/cloud-ex-aws/)
- Docker Swarm
  - [Docker Swarm 1.0 with Multi-host Networking: Manual Setup](http://goelzer.com/blog/2015/12/29/docker-swarmoverlay-networks-manual-method/)
  - [High availability in Docker Swarm](./multi-manager-setup/)
  - [Create a swarm for development](./install-manual/)
  - [Discovery](./discovery/)
- Etcd Discovery Backend
  - [Running etcd under Docker](https://github.com/coreos/etcd/blob/manager/Documentation/docker_guide.md)
  - [Configuration Flags](https://github.com/coreos/etcd/blob/manager/Documentation/configuration.md)
  - [Clustering Guide](https://github.com/coreos/etcd/blob/manager/Documentation/clustering.md)
  - [Running an etcd-Backed Docker Swarm Cluster](http://blog.scottlowe.org/2015/04/19/running-etcd-backed-docker-swarm-cluster/)
  - [Running etcd under Docker](https://coreos.com/etcd/docs/2.0.9/docker_guide.html)
- Networking
  - [Networking](https://docs.docker.com/swarm/networking/)
