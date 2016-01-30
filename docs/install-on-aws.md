<!--[metadata]>
+++
title = "Getting started with Docker Swarm on cloud services"
description = "Deploying Swarm on AWS EC2 AMI's"
keywords = ["docker, swarm, clustering, examples, Amazon, AWS EC2"]
[menu.main]
parent="workw_swarm"
+++
<![end-metadata]-->

# Swarm get started (network pro)

This example shows you how to deploy a high-availability Docker Swarm cluster on  Amazon Web Services (AWS). You'll use Docker Engine on Elastic Cloud (EC2) instances on a Virtual Private Cloud (VPC). Then you'll deploy Swarm managers and nodes to those hosts. During this process you'll implement network security, choose a scheduling strategy, and set up a discovery backend. Finally, you'll deploy a container on your Swarm cluster and learn how high-availability works when one of the managers or nodes goes down.

For a gentler introduction to Swarm, try the [Swarm get started (novice)](https://docs.docker.com/swarm/install-w-machine/) page.

<There are different ways to deploy a Swarm. This topic shows you how to do it ***without using Docker Machine***. For more information about using Docker Machine in a scenario like this one, see: TBD>

<This example doesn't use Amazon's EC2 Container Service (ECS)>

## Prerequisites

- An Amazon Web Services (AWS) account
- Familiarity with AWS products and features:
  - EC2 Virtual Servers in the Cloud
  - Virtual Private Cloud (VPC)
  - VPC Security groups
  - Creating SSH keys and using them to to connect to an EC2 instance


## Step 1: Create an AMI an EC2 instance that has Docker Engine  

1. Log in to your Amazon AWS account and open the Home Console.

2. Open the EC2 Dashboard, create an instance using the *Amazon Linux AMI*.

4. Connect to the instance using SSH.

5. Install Docker Engine on the instance [using these  instructions](https://docs.docker.com/engine/installation/rhel/#install-with-the-script).

    >Make sure you use the `sudo usermod -aG docker ec2-user` command mentioned by the command line output, followed by `logout`. Then log back in and test your D.

6. Now, create an AMI image of your instance.

    ![Create image of EC2 instance](/images/create-image.png)

    Name it something like "AL-AMI running Docker Engine 1.10".

## Create three instances for an etcd discovery backend

<Use information from https://coreos.com/etcd/docs/2.0.9/docker_guide.html#running-a-3-node-etcd-cluster>

## Create two instances for Swarm managers

In the EC2 Dashboard, click *Launch Instance*.

For *Step 1. Choose AMI Image*,  click the tab for *My AMIs*, and select your "AL-AMI running Docker Engine".

![My AMIs Docker Engine](/images/my-amis-select-alami-docker-engine.png)

On *Step 3: Configure Instance Details*, set "Number of Instances" to `2`.

![Configure Instance Details](/images/configure-instance-details.png)

On *Step 5: Tag Instance*, set the `Value` of `Name` to "Swarm manager".

On *Step 6: Configure Security Group*, Use `Create a new security group` to create one called "Swarm managers". Add the following Inbound rules to it:

| Type             | Protocol   | Port Range | Source        |
| ---------------  | -----      | -----      | -----         |
| SSH              | TCP        | 22         | 0.0.0.0/0     |
| HTTP             | TCP        | 80         | 0.0.0.0/0     |
| Custom TCP Rule  | TCP        | 2375       | 172.30.0.0/24 |
| Custom TCP Rule  | TCP        | 3375       | 172.30.0.0/24 |

Then, launch the instances!

## Create two instances for Swarm nodes

Return to the EC2 Dashboard and launch two instances.  

On *Step 5: Tag Instance*, set the `Value` of `Name` to "Swarm node".

On *Step 6: Configure Security Group*, choose "Create a new security group".

Name the security group "Swarm nodes" and add the following Inbound rules to it:

| Type             | Protocol   | Port Range | Source        |
| ---------------  | -----      | -----      | -----         |
| SSH              | TCP        | 22         | 0.0.0.0/0     |
| HTTP             | TCP        | 80         | 0.0.0.0/0     |
| Custom TCP Rule  | TCP        | 2375       | 172.30.0.0/24 |

## Create three instances for etcd discovery backend

Return to the EC2 Dashboard and launch three instances.  

On *Step 5: Tag Instance*, set the `Value` of `Name` to "etcd discovery backend".

On *Step 6: Configure Security Group*, choose "Create a new security group".

Name the security group "etcd discovery backend" and add the following Inbound rules to it:

| Type             | Protocol   | Port Range | Source        |
| ---------------  | -----      | -----      | -----         |
| SSH              | TCP        | 22         | 0.0.0.0/0     |
| Custom TCP Rule  | TCP        | 4001       | 172.30.0.0/24 |
| Custom TCP Rule  | TCP        | 2380       | 172.30.0.0/24 |
| Custom TCP Rule  | TCP        | 2379       | 172.30.0.0/24 |

## Some notes about security and high availability

To use this example in production, consult with an expert to improve your network security. That person may apply more restrictive settings to your security groups, applying a Network Access Control List NACL, and other strategies not mentioned in this example.

For high-availability, you can deploy instances over multiple failure domains, such as AWS availability zones. For example, you can deploy each swarm managers in each of the AWS availability zones for your Region. Refer to the AWS documentation for information about this topic.

## Set up a etcd discovery backend

Here, you're going to create a high-availability discovery backend for Swarm. The etcd nodes are going to run as containers on Docker Engine nodes on three separate EC2 instances.

To start, copy each of the following etcd launch commands to a text document. Replace the <ETCD# PUBLIC IP ADDRESS> placeholders in the text with the publicly visible IP address of each instance.

(To get the IP address, go to EC2 Dashboard > Instances, select an instance, and click the Connect button. This displays the IP address in Connect to Your Instance window.)

![Get public IP address](./images/get-public-ip-address.png)

When you've finished editing the following launch commands, connect to each  instance and use the command to launch an etcd node.

### etcd0 launch command

        docker run -d -p 4001:4001 -p 2380:2380 -p 2379:2379 --name etcd quay.io/coreos/etcd:v2.0.3 \
         -name etcd0 \
         -advertise-client-urls http://<ETCD0 PUBLIC IP ADDRESS>:2379,http://<ETCD0 PUBLIC IP ADDRESS>:4001 \
         -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
         -initial-advertise-peer-urls http://<ETCD0 PUBLIC IP ADDRESS>:2380 \
         -listen-peer-urls http://0.0.0.0:2380 \
         -initial-cluster-token etcd-cluster-1 \
         -initial-cluster etcd0=http://<ETCD0 PUBLIC IP ADDRESS>:2380,etcd1=http://<ETCD1 PUBLIC IP ADDRESS>:2380,etcd2=http://<ETCD2 PUBLIC IP ADDRESS>:2380 \
         -initial-cluster-state new

### etcd1

        docker run -d -p 4001:4001 -p 2380:2380 -p 2379:2379 --name etcd quay.io/coreos/etcd:v2.0.3 \
         -name etcd1 \
         -advertise-client-urls http://<ETCD1 PUBLIC IP ADDRESS>:2379,http://<ETCD1 PUBLIC IP ADDRESS>:4001 \
         -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
         -initial-advertise-peer-urls http://<ETCD1 PUBLIC IP ADDRESS>:2380 \
         -listen-peer-urls http://0.0.0.0:2380 \
         -initial-cluster-token etcd-cluster-1 \
         -initial-cluster etcd0=http://<ETCD0 PUBLIC IP ADDRESS>:2380,etcd1=http://<ETCD1 PUBLIC IP ADDRESS>:2380,etcd2=http://<ETCD2 PUBLIC IP ADDRESS>:2380 \
         -initial-cluster-state new

### etcd2

        docker run -d -p 4001:4001 -p 2380:2380 -p 2379:2379 --name etcd quay.io/coreos/etcd:v2.0.3 \
         -name etcd2 \
         -advertise-client-urls http://<ETCD2 PUBLIC IP ADDRESS>:2379,http://<ETCD2 PUBLIC IP ADDRESS>:4001 \
         -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
         -initial-advertise-peer-urls http://<ETCD2 PUBLIC IP ADDRESS>:2380 \
         -listen-peer-urls http://0.0.0.0:2380 \
         -initial-cluster-token etcd-cluster-1 \
         -initial-cluster etcd0=http://<ETCD0 PUBLIC IP ADDRESS>:2380,etcd1=http://<ETCD1 PUBLIC IP ADDRESS>:2380,etcd2=http://<ETCD2 PUBLIC IP ADDRESS>:2380 \
         -initial-cluster-state new

<For more information, see https://coreos.com/etcd/docs/2.0.9/docker_guide.html#running-a-3-node-etcd-cluster>

## Create the Swarm managers

Connect to each of the instances named "Swarm manager" using SSH.

Start the Swarm manager.

        docker run -d -p <manager_port>:2375 swarm manage etcd://<ip1>,<ip2>/<path>


The manager is exposed and listening on `<manager_port>`.

Check your configuration by running `docker info` as follows:

		docker -H tcp://<manager_ip:manager_port> info

	For example, if you run the manager locally on your machine:

		$ docker -H tcp://0.0.0.0:2375 info
		Containers: 0
		Nodes: 3
		 agent-2: 172.31.40.102:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB
		 agent-1: 172.31.40.101:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB
		 agent-0: 172.31.40.100:2375
			└ Containers: 0
			└ Reserved CPUs: 0 / 1
			└ Reserved Memory: 0 B / 514.5 MiB

<If you are running a test cluster without TLS enabled, you may get an error. If so, unset `DOCKER_TLS_VERIFY` with:

    $ unset DOCKER_TLS_VERIFY>

<update procedure to use TLS>    

### Talk about ports
Open ports
Talk about port options
Specific discussion of difference between ports 2375, 2376 and 3375/6

## Create the Swarm nodes


## Connect to the Swarm master and deploy a container on a swarm
Explain the relationship via text/diagram
Illustrate via command-line examples how to do this
Leverage portions of getting started (novice) example.

## More resources

- Deploying a Swarm cluster on AWS raises the question of whether to add more EC2 instances or increase the CPU/RAM of each instance. Nati Shalom sheds light on this complex question in his [http://natishalom.typepad.com/nati_shaloms_blog/2010/09/scale-up-vs-scale-out.html](Scale-out vs Scale-up) blog post.
- Check out Mike Goelzer's [https://github.com/mgoelzer/swarm-demo-voting-app](Swarm Example Cluster: Web App) on GitHub.
