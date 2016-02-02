<!--[metadata]>
+++
title = "Build a Swarm cluster for production"
description = "Deploying Swarm on AWS EC2 AMI's in a VPC"
keywords = ["docker, swarm, clustering, examples, Amazon, AWS EC2"]
[menu.main]
parent="workw_swarm"
weight=-40
+++
<![end-metadata]-->


# Build a Swarm cluster for production

This example shows you how to deploy a high-availability Docker Swarm cluster. Although this example uses the Amazon Web Services (AWS) platform, you can deploy an equivalent Docker Swarm cluster on many other platforms.

The Swarm cluster will contain three types of nodes:
- Swarm manager
- Swarm node (aka Swarm agent)
- Discovery backend node running consul

This example will take you through the following steps: You establish basic network security by creating a security group that restricts inbound traffic by port number, type, and origin. Then, you create four hosts on your network by launching Elastic Cloud (EC2) instances, applying the appropriate security group to each one, and installing Docker Engine on each one. You create a discovery backend by running an consul container on one of the hosts. You create the Swarm cluster by running two Swarm managers in a high-availability configuration. One of the Swarm managers shares a host with consul. Then you run two Swarm nodes. You communicate with the Swarm via the primary manager, running a simple hello world application and then checking which node ran the application. To finish, you test high-availability by making one of Swarm managers fail and checking the status of the managers.

For a gentler introduction to Swarm, try the [Evaluate Swarm in a sandbox](install-w-machine) page.

<This example doesn't use Amazon's EC2 Container Service (ECS)>

## Prerequisites

- An Amazon Web Services (AWS) account
- Familiarity with AWS features and tools, such as:
  - Elastic Cloud (EC2) Dashboard
  - Virtual Private Cloud (VPC) Dashboard
  - VPC Security groups
  - Connecting to an EC2 instance using SSH

## Update the network security rules

AWS uses a "security group" to allow specific types of network traffic on your VPC network. The **default** security group's initial set of rules deny all inbound traffic, allow all outbound traffic, and allow all traffic between instances. You're  going to add a couple of rules to allow inbound SSH connections and inbound container images. This set of rules somewhat protects the Engine, Swarm, and Consul ports. For a production environment, you would apply more restrictive security measures. Do not leave Docker Engine ports unprotected.

From your AWS home console, click **VPC - Isolated Cloud Resources**. Then, in the VPC Dashboard that opens, navigate to **Security Groups**. Select the **default** security group that's associated with your default VPC and add the following two rules. (The **Allows** column is just for your reference.)

| Type            | Protocol  | Port Range | Source        | Allows            |
| --------------  | -----     | -----      | -----         | -----             |
| SSH             | TCP       | 22         | 0.0.0.0/0     | SSH connection    |
| HTTP            | TCP       | 80         | 0.0.0.0/0     | Container images  |


## Create your hosts

Here, you create five Linux hosts that are part of the "Docker Swarm Example" security group.

Open the EC2 Dashboard and launch four EC2 instances, one at a time:

- During **Step 1: Choose an Amazon Machine Image (AMI)**, pick the *Amazon Linux AMI*.

- During **Step 5: Tag Instance**, under **Value**, give each instance one of these names:
    - manager0 & consul0
    - manager1
    - node0
    - node1

- During **Step 6: Configure Security Group**, choose **Select an existing security group** and pick the "default" security group.

Review and launch your instances.

## Install Docker Engine on each instance

Connect to each instance using SSH and install Docker Engine.

Update the yum packages, and keep an eye out for the "y/n/abort" prompt:

        $ sudo yum update

Run the installation script:

        $ curl -sSL https://get.docker.com/ | sh

Configure and start Docker Engine so it listens for Swarm nodes on port 2375 :

        $ sudo docker daemon -H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock

Verify that Docker Engine is installed correctly:

        $ sudo docker run hello-world

The output should display a "Hello World" message and other text without any error messages.

Give the ec2-user root privileges:

        sudo usermod -aG docker ec2-user

 Then, enter `logout`.

> Troubleshooting: If entering a `docker` command produces a message asking whether docker is available on this host, it may be because the user doesn't have root privileges. If so, use `sudo` or give the user root privileges.
> For this example, don't create an AMI image from one of your instances running Docker Engine and then re-use it to create the other instances. Doing so will produce errors.
> Troubleshooting: If your host cannot reach Docker Hub, the `docker run` commands that pull container images may fail. In that case, check that your VPC is associated with a security group with a rule that allows inbound traffic (e.g., HTTP/TCP/80/0.0.0.0/0). Also Check
the [Docker Hub status page](http://status.docker.com/) for service
availability.

## Set up an consul discovery backend

Here, you're going to create a minimalist discovery backend. The Swarm managers and nodes use this backend to authenticate themselves as members of the cluster. The Swarm managers also use this information to identify which nodes are available to run containers.

To keep things simple, you are going to run a single consul daemon on the same host as one of the Swarm managers.

To start, copy the following launch command to a text file.

        $ docker run -d -p 8500:8500 —name=consul progrium/consul -server -bootstrap

Then, use SSH to connect to the "manager0 & consul0" instance. At the command line, enter `ifconfig`. From the output, copy the `eth0` IP address from `inet addr`.

Using SSH, connect to the "manager0 & etc0" instance. Copy the launch command from the text file and paste it into the command line.

Your consul node is up and running, providing your cluster with a discovery backend. To increase its reliability, you can create a high-availability cluster using a trio of consul nodes using the link mentioned at the end of this page. (Before creating a cluster of console nodes, update the VPC security group with rules to allow inbound traffic on the required port numbers.)

## Create a high-availability Swarm cluster

After creating the discovery backend, you can create the Swarm managers. Here, you are going to create two Swarm managers in a high-availability configuration. The first manager you run becomes the Swarm's *primary manager*. Some documentation still refers to a primary manager as a "master", but that term has been superseded. The second manager you run serves as a *replica*. If the primary manager becomes unavailable, the cluster elects the replica as the primary manager.

To create the primary manager in a high-availability Swarm cluster, use the following syntax:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise <manager0_ip>:4000  consul://<consul_ip>

Because this is particular manager is on the same "manager0 & consul0" instance as the consul node, replace both `<manager0_ip>` and `<consul_ip>` with the same IP address. For example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000  consul://172.30.0.161:8500

Enter `docker ps`. From the output, verify that both a swarm and an consul container are running. Then, disconnect from the "manager0 & consul0" instance.

Connect to the "manager1" instance and use `ifconfig` to get its IP address. Then, enter the following command, replacing `<manager1_ip>`. For example:

        $ docker run -d swarm manage -H :4000 --replication --advertise <manager1_ip>:4000  consul://172.30.0.161:8500

Enter `docker ps` and, from the output, verify that a swarm container is running.

Now, connect to each of the "node0" and "node1" instances, get their IP addresses, and run a Swarm node on each one using the following syntax:

        $ docker run -d swarm join --advertise=<node_ip>:2375 consul://<consul_ip>:8500

For example:

        $ docker run -d swarm join --advertise=172.30.0.69:2375 consul://172.30.0.161:8500


Your small Swarm cluster is up and running on multiple hosts, providing you with a high-availability virtual Docker Engine. To increase its reliability and capacity, you can add more Swarm managers, nodes, and a high-availability discovery backend.

## Communicate with the Swarm

You can communicate with the Swarm to get information about the managers and nodes using the Swarm API, which is nearly the same as the standard Docker API.  
In this example, you use SSL to connect to "manager0 & etc0" host again. Then, you address commands to the Swarm manager.

Get information about the master and nodes in the cluster:

        $ docker -H :4000 info

The output gives the manager's role as primary (`Role: primary`) and information about each of the nodes.

Now run an application on the Swarm:

        $ docker -H :4000 run hello-world

Check which Swarm node ran the application:

        $ docker -H :4000 ps

## Test the high-availability Swarm managers

To see the replica instance take over, you're going to shut down the primary manager. Doing so kicks off an election, and the replica becomes the primary manager. When you start the manager you shut down earlier, it becomes the replica.

Using an SSH connection to the "manager0 & etc0" instance, get the container id or name of the swarm container:

        $ docker ps

Shut down the primary master, replacing `<id_name>` with the container id or name (e.g., "8862717fe6d3" or "trusting_lamarr").

        $ docker rm -f <id_name>

Start the swarm master. For example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000  consul://172.30.0.161:237

Look at the logs, replacing `<id_name>` with the new container id or name:

       $ sudo docker logs <id_name>

The output shows will show two entries like these ones:

        time="2016-02-02T02:12:32Z" level=info msg="Leader Election: Cluster leadership lost"
        time="2016-02-02T02:12:32Z" level=info msg="New leader elected: 172.30.0.160:4000"

To get information about the master and nodes in the cluster, enter:

        $ docker -H :4000 info

You can connect to the "master1" node and run the `info` and `logs` commands. They will display corresponding entries for the change in leadership.

## Additional Resources

- Installing Docker Engine
    - [Example: Manual install on a cloud provider](http://docs.docker.com/engine/installation/cloud/cloud-ex-aws/)
- Docker Swarm
  - [Docker Swarm 1.0 with Multi-host Networking: Manual Setup](http://goelzer.com/blog/2015/12/29/docker-swarmoverlay-networks-manual-method/)
  - [High availability in Docker Swarm](multi-manager-setup/)
  - [Discovery](discovery/)
- consul Discovery Backend
  - [high-availability cluster using a trio of consul nodes](https://hub.docker.com/r/progrium/consul/)
- Networking
  - [Networking](https://docs.docker.com/swarm/networking/)
