---
advisory: swarm-standalone
hide_from_sitemap: true
description: Deploying Swarm on AWS EC2 AMI's in a VPC
keywords: docker, swarm, clustering, examples, Amazon, AWS EC2
title: Build a Swarm cluster for production
---

This page teaches you to deploy a high-availability swarm cluster.
Although the example installation uses the Amazon Web Services (AWS) platform,
you can deploy an equivalent swarm on many other platforms. In this example, you do the following:

- [Verify you have the prerequisites](install-manual.md#prerequisites)
- [Establish basic network security](install-manual.md#step-1-add-network-security-rules)
- [Create your nodes](install-manual.md#step-2-create-your-instances)
- [Install Engine on each node](install-manual.md#step-3-install-engine-on-each-node)
- [Configure a discovery backend](install-manual.md#step-4-set-up-a-discovery-backend)
- [Create a swarm cluster](install-manual.md#step-5-create-swarm-cluster)
- [Communicate with the swarm](install-manual.md#step-6-communicate-with-the-swarm)
- [Test the high-availability swarm managers](install-manual.md#step-7-test-swarm-failover)
- [Additional Resources](install-manual.md#additional-resources)

For a quickstart for Docker Swarm, try the [Evaluate Swarm in a sandbox](install-w-machine.md) page.

## Prerequisites

- An Amazon Web Services (AWS) account
- Familiarity with AWS features and tools, such as:
  - Elastic Cloud (EC2) Dashboard
  - Virtual Private Cloud (VPC) Dashboard
  - VPC Security groups
  - Connecting to an EC2 instance using SSH

## Step 1. Add network security rules

AWS uses a "security group" to allow specific types of network traffic on your
VPC network. The **default** security group's initial set of rules deny all
inbound traffic, allow all outbound traffic, and allow all traffic between
instances.

You're going to add a couple of rules to allow inbound SSH connections and
inbound container images. This set of rules somewhat protects the Engine, Swarm,
and Consul ports. For a production environment, you would apply more restrictive
security measures. Do not leave Docker Engine ports unprotected.

From your AWS home console, do the following:

1. Click **VPC - Isolated Cloud Resources**.

    The VPC Dashboard opens.

2. Navigate to **Security Groups**.

3. Select the **default** security group that's associated with your default VPC.

4. Add the following two rules.

    <table>
    <tr>
      <th>Type</th>
      <th>Protocol</th>
      <th>Port Range</th>
      <th>Source</th>
    </tr>
    <tr>
      <td>SSH</td>
      <td>TCP</td>
      <td>22</td>
      <td>0.0.0.0/0</td>
    </tr>
    <tr>
      <td>HTTP</td>
      <td>TCP</td>
      <td>80</td>
      <td>0.0.0.0/0</td>
    </tr>
    </table>

  The SSH connection allows you to connect to the host while the HTTP is for container images.

## Step 2. Create your instances

In this step, you create five Linux hosts that are part of your default security
group. When complete, the example deployment contains three types of nodes:

| Node Description                     | Name                    |
|--------------------------------------|-------------------------|
| Swarm primary and secondary managers | `manager0`, `manager1` |
| Swarm node                           | `node0`, `node1`        |
| Discovery backend                    | `consul0`               |

To create the instances do the following:

1. Open the EC2 Dashboard and launch five EC2 instances, one at a time.

    - During **Step 1: Choose an Amazon Machine Image (AMI)**, pick the *Amazon Linux AMI*.

    - During **Step 5: Tag Instance**, under **Value**, give each instance one of these names:

        - `manager0`
        - `manager1`
        - `consul0`
        - `node0`
        - `node1`

    - During **Step 6: Configure Security Group**, choose **Select an existing security group** and pick the "default" security group.

2. Review and launch your instances.

## Step 3. Install Engine on each node

1.  [Install Docker](../engine/install/index.md){: target="_blank" class="_"}
    on each host, using the appropriate instructions for your operating system
    and distribution.

2.  Edit `/etc/docker/daemon.json`. Create it if it does not exist. Assuming the
    file was empty, its contents should be:

    ```json
    {
      "hosts": ["tcp://0.0.0.0:2375", "unix:///var/run/docker.sock"]
    }
    ```

    Start or restart Docker for the changes to take effect.

    ```bash
    $ sudo systemctl start docker
    ```

3.  Give the `ec2-user` root privileges:

    ```bash
    $ sudo usermod -aG docker ec2-user
    ```

4. Log out of the host.

#### Troubleshooting

* If entering a `docker` command produces a message asking whether docker is
available on this host, it may be because the user doesn't have root privileges.
If so, use `sudo` or give the user root privileges.

* For this example, don't create an AMI image from one of your instances running
Docker Engine and then re-use it to create the other instances. Doing so
produces errors.

* If your host cannot reach Docker Hub, `docker run` commands that pull
images fail. In that case, check that your VPC is associated with
a security group with a rule that allows inbound traffic. Also check the [Docker Hub status
page](http://status.docker.com/) for service availability.

## Step 4. Set up a discovery backend

Here, you're going to create a minimalist discovery backend. The swarm managers
and nodes use this backend to authenticate themselves as members of the cluster.
The swarm managers also use this information to identify which nodes are
available to run containers.

To keep things simple, you are going to run a single consul daemon on the same
host as one of the swarm managers.

1. Use SSH to connect to the `consul0` instance.

        $ ifconfig

2. From the output, copy the `eth0` IP address from `inet addr`.

3. To set up a discovery backend, use the following command, replacing `<consul0_ip>` with the IP address from the previous command:

        $ docker run -d -p 8500:8500 --name=consul progrium/consul -server -bootstrap -advertise=<consul0_ip>

4. Enter `docker ps`.

    From the output, verify that a consul container is running.
    Then, disconnect from the `consul0` instance.

Your Consul node is up and running, providing your cluster with a discovery
backend. To increase its reliability, you can create a high-availability cluster
using a trio of consul nodes using the link mentioned at the end of this page.
(Before creating a cluster of consul nodes, update the VPC security group with
rules to allow inbound traffic on the required port numbers.)

## Step 5. Create swarm cluster

After creating the discovery backend, you can create the swarm managers. In this step, you are going to create two swarm managers in a high-availability configuration. The first manager you run becomes the swarm's *primary manager*. Some documentation still refers to a primary manager as a "master", but that term has been superseded. The second manager you run serves as a *replica*. If the primary manager becomes unavailable, the cluster elects the replica as the primary manager.

1. Use SSH to connect to the `manager0` instance and use `ifconfig` to get its IP address.

        $ ifconfig

2. To create the primary manager in a high-availability swarm cluster, use the following syntax:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise <manager0_ip>:4000 consul://<consul0_ip>:8500

    Replacing `<manager0_ip>` and `<consul0_ip>` with the IP address from the previous command, for example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.125:4000 consul://172.30.0.161:8500

3. Enter `docker ps`.

    From the output, verify that a swarm cluster container is running.
    Then, disconnect from the `manager0` instance.

4. Connect to the `manager1` node and use `ifconfig` to get its IP address.

        $ ifconfig

5. Start the secondary swarm manager using following command.

      Replacing `<manager1_ip>` with the IP address from the previous command, for example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise <manager1_ip>:4000 consul://172.30.0.161:8500

6. Enter `docker ps` to verify that a swarm container is running. Then disconnect from the `manager1` instance.

7. Connect to `node0` and `node1` in turn and join them to the cluster.

    a. Get the node IP addresses with the `ifconfig` command.

    b. Start a swarm container each using the following syntax:

        docker run -d swarm join --advertise=<node_ip>:2375 consul://<consul0_ip>:8500

      For example:

        $ docker run -d swarm join --advertise=172.30.0.69:2375 consul://172.30.0.161:8500

    c. Enter `docker ps` to verify that the swarm cluster container started from the previous command is running.

Your small swarm cluster is up and running on multiple hosts, providing you with a high-availability virtual Docker Engine. To increase its reliability and capacity, you can add more swarm managers, nodes, and a high-availability discovery backend.

## Step 6. Communicate with the swarm

You can communicate with the swarm to get information about the managers and
nodes using the Swarm API, which is nearly the same as the standard Docker API.
In this example, you use SSL to connect to `manager0` and `consul0` host again.
Then, you address commands to the swarm manager.

1. Get information about the manager and nodes in the cluster:

        $ docker -H :4000 info

    The output gives the manager's role as primary (`Role: primary`) and
    information about each of the nodes.

2. Run an application on the swarm:

        $ docker -H :4000 run hello-world

3. Check which swarm node ran the application:

        $ docker -H :4000 ps

## Step 7. Test Swarm failover

To see the replica instance take over, you're going to shut down the primary
manager. Doing so kicks off an election, and the replica becomes the primary
manager. When you start the manager you shut down earlier, it becomes the
replica.

1. SSH connection to the `manager0` instance.

2. Get the container ID or name of the `swarm` container:

        $ docker ps

3. Shut down the primary manager, replacing `<id_name>` with the container's ID or name (for example, "8862717fe6d3" or "trusting_lamarr").

        docker container rm -f <id_name>

4. Start the swarm manager. For example:

        $ docker run -d -p 4000:4000 swarm manage -H :4000 --replication --advertise 172.30.0.161:4000 consul://172.30.0.161:8500

5. Review the Engine's daemon logs, replacing `<id_name>` with the new container's ID or name:

        $ sudo docker logs <id_name>

    The output shows two entries like these ones:

        time="2016-02-02T02:12:32Z" level=info msg="Leader Election: Cluster leadership lost"
        time="2016-02-02T02:12:32Z" level=info msg="New leader elected: 172.30.0.160:4000"

6. To get information about the manager and nodes in the cluster, enter:

        $ docker -H :4000 info

You can connect to the `manager1` node and run the `info` and `logs` commands.
They display corresponding entries for the change in leadership.

## Additional resources

- [High availability in Docker Swarm](multi-manager-setup.md)
- [Discovery](discovery.md)
- [High-availability cluster using a trio of consul nodes](https://hub.docker.com/r/progrium/consul/)
- [Networking](networking.md)
