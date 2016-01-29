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

- An Amazon AWS account
- Familiarity with:
  - AWS services and procedures
  - Using SSH to connect to an EC2 instance

## Step 1: Create an AMI an EC2 instance running Docker Engine  

1. Log in to your Amazon AWS account and open the Home Console.

2. Open the EC2 Dashboard, create an instance using the *Amazon Linux AMI*.

2. Connect to the instance using SSH.

3. Install Docker Engine [using these Amazon instructions](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/docker-basics.html#install_docker).

    >Ignore the note in Amazon's instructions about yum not installing the minimum Docker version of 1.5.0. At the time this is being written, running the `sudo yum install -y docker` command installs the recommended version, 1.9.1.

    <To install the latest version of Docker Engine, do xy...>

3. Instead of repeating the previous steps for every EC2 instance,  [http://docs.aws.amazon.com/AWSToolkitVS/latest/UserGuide/tkv-create-ami-from-instance.html](create an AMI image of your instance). Name it something like "AL-AMI running Docker Engine" and include the version of Docker Engine in the description.

    While AWS finishes generating the AMI, go create security groups for your instances. When you're done with that, you'll use your new AMI to create multiple EC2 instances.

## Create security groups

Create three security groups that you can use to control inbound traffic for each type of instance.  

In your AWS **EC2 Dashboard**, under **NETWORK & SECURITY > Security Groups**, create three security groups with the following inbound settings. (You can leave the outbound settings unchanged.)

Important: For the entries that start with **Custom TCP Rule**, don't use the VPC Classless Inter-Domain Routing (CIDR) block shown in the following screen shots under **Source**.  Instead, use the CIDR block for *the VPC your instance is using*.

>To obtain the CIDR block of your VPC: Look at the description of your instance, and get its VPC ID. Then, from the Home Console, open the VPC dashboard, and look at the VPC that uses the same ID, and get its VPC CIDR block.

> If you assign your security groups to a VPC (optional), assign them to the same VPC that the EC2 instances belong to.

> Unless you purposefully created a VPC for this example, the EC2 instance probably belongs to the *default* VPC security group.

### Swarm mangers

![Swarm managers](/images/security-group-inbound-rules-for-managers.png)

### Swarm nodes

![Swarm managers](/images/security-group-inbound-rules-for-nodes.png)

### Consul

![Swarm managers](/images/security-group-inbound-rules-for-consul.png)

## Create EC2 instances using your AMI

In the EC2 Dashboard, click *Launch Instance*. For *Step 1. Choose AMI Image*,  click the tab for *My AMIs*, and select your "AL-AMI running Docker Engine".

![My AMIs Docker Engine](/images/my-amis-select-alami-docker-engine.png)

On *Step 3: Configure Instance Details*, change the "Number of Instances". For this example, you only need two instances, but feel free to create more.

![Configure Instance Details](/images/configure-instance-details.png)

On *Step 6: Configure Security Group*, choose "Select an existing security group" and change the security group so only **Swarm managers** is selected.

![Security Group for Swarm Managers](/images/configure-security-group-manager.png)

Launch the instances that will become Swarm managers!

Return to the EC2 Dashboard and repeat these steps, launching two instances that use security group for **Swarm nodes**.    

![Security Group for Swarm Nodes](/images/configure-security-group-nodes.png)

## Some notes about security and high availability

For production, consult with a security expert to tighten your network security. This can include applying more restrictive settings to your security groups, applying a Network Access Control List NACL, and other strategies not mentioned in this example.

For high-availability, you can deploy instances over multiple failure domains, such as AWS availability zones. For example, you can deploy each swarm managers in each of the AWS availability zones for your Region. Refer to the AWS documentation for information about this topic.

## Create the Swarm managers

Back in the EC2 Dashboard, under Instances, use the filter to display only instances whose Security Group Name is Swarm Managers.

![Filter by Security Group for Swarm Managers](/images/filter-instance-by-security-group-manager.png)

***For each instance***, do the following.

Connect using SSH.

![Connect to each manager instance using SSH](/images/connect-to-a-manager.png)

        rolfedlugy-hegwer at RsMBP in ~/Downloads
        $ ssh -i "macamikey.pem" root@52.91.210.70
        The authenticity of host '52.91.210.70 (52.91.210.70)' can't be established.
        ECDSA key fingerprint is SHA256:EUeUxxasdfzEafUz9ffadsKfBGvasHl/afdWXZXafdU.
        Are you sure you want to continue connecting (yes/no)?

Start the Swarm manager.

        docker run -d -p <manager_port>:2375 swarm manage token://<cluster_id>

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

## Step 2: Set up your EC2 network
Open ports
Talk about port options
Specific discussion of difference between ports 2375, 2376 and 3375/6

## Step 3.  Create a Discovery Token
Get a token for the cluster
Explain what the token is for

## Step 4: Create the HA Swarm managers
Create the managers including TCP ports and Discovery

## Step 5. Create the Swarm nodes

## Step 6. Deploy a container on a swarm
Explain the relationship via text/diagram
Illustrate via command-line examples how to do this
Leverage portions of getting started (novice) example.

## More resources

- Deploying a Swarm cluster on AWS raises the question of whether to add more EC2 instances or increase the CPU/RAM of each instance. Nati Shalom sheds light on this complex question in his [http://natishalom.typepad.com/nati_shaloms_blog/2010/09/scale-up-vs-scale-out.html](Scale-out vs Scale-up) blog post.
- Check out Mike Goelzer's [https://github.com/mgoelzer/swarm-demo-voting-app](Swarm Example Cluster: Web App) on GitHub.
