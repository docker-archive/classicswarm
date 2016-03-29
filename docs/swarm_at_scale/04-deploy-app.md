<!--[metadata]>
+++
title = "Deploy the application"
description = "Try Swarm at scale"
keywords = ["docker, swarm, scale, voting, application,  certificates"]
[menu.main]
parent="scale_swarm"
weight=-80
+++
<![end-metadata]-->

# Deploy the application

You've [built a Swarm cluster](03-create-cluster.md) so now you are ready to
build and deploy the voting application itself.


## Step 1: Learn about the images

Some of the application's containers are launched form existing images pulled
directly from Docker Hub. Other containers are launched from custom images you
must build. The list below shows which containers use custom images and which do
not:

- Load balancer container: stock image (`ehazlett/interlock`)
- Redis containers: stock image (official `redis` image)
- Postgres (PostgreSQL) containers: stock image (official `postgres` image)
- Web containers: custom built image
- Worker containers: custom built image
- Results containers: custom built image

All custom built images are built using Dockerfiles pulled from the
[example application's public GitHub
repository](https://github.com/docker/swarm-microservice-demo-v1).

1. If you haven't already, `ssh` into the Swarm `manager` node.

2. Clone the [application's GitHub repo](https://github.com/docker/swarm-microservice-demo-v1)

        $ git clone https://github.com/docker/swarm-microservice-demo-v1
        sudo: unable to resolve host master
        Cloning into 'swarm-microservice-demo-v1'...
        remote: Counting objects: 304, done.
        remote: Compressing objects: 100% (17/17), done.
        remote: Total 304 (delta 5), reused 0 (delta 0), pack-reused 287
        Receiving objects: 100% (304/304), 2.24 MiB | 2.88 MiB/s, done.
        Resolving deltas: 100% (132/132), done.
        Checking connectivity... done.

    This command creates a new directory structure inside of your working
    directory. The new directory contains all of the files and folders required
    to build the voting application images.

    The `AWS` directory contains the `cloudformation.json` file used to deploy
    the EC2 instances. The `Vagrant` directory contains files and instructions
    required to deploy the application using Vagrant. The `results-app`,
    `vote-worker`, and `web-vote-app` directories contain the Dockerfiles and
    other files required to build the custom images for those particular
    components of the application.

3. Change directory into the `swarm-microservice-demo-v1/web-vote-app` directory.

        $ cd swarm-microservice-demo-v1/web-vote-app/

4. View the Dockerfile contents.

        $ cat Dockerfile
        # Using official python runtime base image
        FROM python:2.7
        # Set the application directory
        WORKDIR /app
        # Install our requirements.txt
        ADD requirements.txt /app/requirements.txt
        RUN pip install -r requirements.txt
        # Copy our code from the current folder to /app inside the container
        ADD . /app
        # Make port 80 available for links and/or publish
        EXPOSE 80
        # Define our command to be run when launching the container
        CMD ["python", "app.py"]

    As you can see, the image is based on the official `Python:2.7` tagged
    image, adds a requirements file into the `/app` directory, installs
    requirements, copies files from the build context into the container,
    exposes port `80` and tells the container which command to run.

5. Spend time investigating the other parts of the application by viewing the `results-app/Dockefile` and the `vote-worker/Dockerfile` in the application.

## Step 2. Build custom images

1. If you haven't already, `ssh` into the Swarm `manager` node.

2. Make sure you have DOCKER_HOST set

        $ export DOCKER_HOST="tcp://192.168.33.11:3375"

3. Change to the root of your `swarm-microservice-demo-v1` clone.

4. Build the `web-votes-app` image both the front end nodes.

    **frontend01**:

        $ docker -H tcp://192.168.33.20:2375 build -t web-vote-app ./web-vote-app

    **frontend02**:

        $ docker -H tcp://192.168.33.21:2375 build -t web-vote-app ./web-vote-app

    These commands build the  `web-vote-app` image on the `frontend01` and
    `frontend02` nodes. To accomplish the operation, each command copies the
    contents of the `swarm-microservice-demo-v1/web-vote-app` sub-directory from the
    `manager` node to each frontend node. The command then instructs the
    Docker daemon on each frontend node to build the image and store it locally.

    You'll notice this example uses a `-H` flag to pull an image to specific
    host. This is to help you conceptualize the architecture for this sample. In
    a production deployment, you'd omit this option and rely on the Swarm
    manager to distribute the image. The manager would pull the image to every
    node; so that any node can step in to run the image as needed.

    It may take a minute or so for each image to build. Wait for the builds to finish.

5. Build `vote-worker` image on the `worker01` node

        $ docker -H tcp://192.168.33.200:2375 build -t vote-worker ./vote-worker

    It may take a minute or so for the image to build. Wait for the build to
    finish.

5. Build the `results-app` on the `store` node

        $ docker -H tcp://192.168.33.250:2375 build -t results-app ./results-app

Each of the *custom images* required by the application is now built and stored
locally on the nodes that will use them.

## Step 3. Pull images from Docker Hub

For performance reasons, it is always better to pull any required Docker Hub
images locally on each instance that needs them. This ensures that containers
based on those images can start quickly.

1. Log into the Swarm `manager` node.

2. Pull the `redis` image to your frontend nodes.

    **frontend01**:

        $ docker -H tcp://192.168.33.20:2375 pull redis

    **frontend02**:

        $ docker -H tcp://192.168.33.21:2375 pull redis

2. Pull the `postgres` image to the `store` node

        $ docker -H tcp://192.168.33.250:2375 pull postgres

3. Pull the `ehazlett/interlock` image to the `interlock` node

        $ docker -H tcp://192.168.33.12:2375 pull ehazlett/interlock

Each node in the cluster, as well as the `interlock` node, now has the required images stored locally as shown below.

![](../images/interlock.jpg)

Now that all images are built, pulled, and stored locally, the next step is to start the application.

## Step 4. Start the voting application

In the following steps, your launch several containers to the voting application.

1. If you haven't already, `ssh` into the Swarm `manager` node.

2. Start the `interlock` container on the `interlock` node

        $ docker -H tcp://192.168.33.12:2375 run --restart=unless-stopped -p 80:80 --name interlock -d ehazlett/interlock --swarm-url tcp://192.168.33.11:3375 --plugin haproxy start

    This command is issued against the `interlock` instance and maps port 80 on the instance to port 80 inside the container. This allows the container to load balance connections coming in over port 80 (HTTP). The command also applies the `--restart=unless-stopped` policy to the container, telling Docker to restart the container if it exits unexpectedly.

3. Verify the container is running.

        $ docker -H tcp://192.168.33.12:2375 ps

4. Start a `redis` container on your front end nodes.

    **frontend01**:

        $ docker run --restart=unless-stopped --env="constraint:node==frontend01" -p 6379:6379 --name redis01 --net mynet -d redis
        $ docker -H tcp://192.168.33.20:2375 ps

    **frontend02**:

        $ docker run --restart=unless-stopped --env="constraint:node==frontend02" -p 6379:6379 --name redis02 --net mynet -d redis
        $ docker -H tcp://192.168.33.21:2375 ps

    These two commands are issued against the Swarm cluster. The commands specify *node constraints*, forcing Swarm to start the containers on `frontend01` and `frontend02`. Port 6379 on each instance is mapped to port 6379 inside of each container for debugging purposes. The command also applies the `--restart=unless-stopped` policy to the containers and attaches them to the `mynet` overlay network.

5. Start a `web-vote-app` container the frontend nodes.

    **frontend01**:

        $ docker run --restart=unless-stopped --env="constraint:node==frontend01" -d -p 5000:80 -e WEB_VOTE_NUMBER='01' --name frontend01 --net mynet --hostname votingapp.local web-vote-app

    **frontend02**:

        $ docker run --restart=unless-stopped --env="constraint:node==frontend02" -d -p 5000:80 -e WEB_VOTE_NUMBER='02' --name frontend02 --net mynet --hostname votingapp.local web-vote-app

    These two commands are issued against the Swarm cluster. The commands
    specify *node constraints*, forcing Swarm to start the containers on
    `frontend01` and `frontend02`. Port `5000` on each node is mapped to port
    `80` inside of each container. This allows connections to come in to each
    node on port `5000` and be forwarded to port `80` inside of each container.

    Both containers are attached to the `mynet` overlay network and both
    containers are given the `votingapp-local` hostname. The
    `--restart=unless-stopped` policy is also applied to these containers.

4. Start the `postgres` container on the `store` node

        $ docker run --restart=unless-stopped --env="constraint:node==store" --name pg -e POSTGRES_PASSWORD=pg8675309 --net mynet -p 5432:5432 -d postgres

    This command is issued against the Swarm cluster and starts the container on
    `store`. It maps port 5432 on the `store` node to port 5432 inside the
    container and attaches the container to the `mynet` overlay network. The
    command also inserts the database password into the container via the
    `POSTGRES_PASSWORD` environment variable and applies the
    `--restart=unless-stopped` policy to the container.

     Sharing passwords like this is not recommended for production use cases.

5. Start the `worker01` container on the `worker01` node

        $ docker run --restart=unless-stopped --env="constraint:node==worker01" -d -e WORKER_NUMBER='01' -e FROM_REDIS_HOST=1 -e TO_REDIS_HOST=2 --name worker01 --net mynet vote-worker

    This command is issued against the Swarm manager and uses a constraint to
    start the container on the `worker01` node. It passes configuration data
    into the container via environment variables, telling the worker container
    to clear the queues on `frontend01` and `frontend02`. It adds the container
    to the `mynet` overlay network and applies the `--restart=unless-stopped`
    policy to the container.

6. Start the `results-app` container on the `store` node

        $ docker run --restart=unless-stopped --env="constraint:node==store" -p 80:80 -d --name results-app --net mynet results-app

    This command starts the results-app container on the `store` node by means
    of a *node constraint*. It maps port 80 on the `store` node to port 80
    inside the container. It adds the container to the `mynet` overlay network
    and applies the `--restart=unless-stopped` policy to the container.

The application is now fully deployed as shown in the diagram below.

![](../images/fully-deployed.jpg)

## Step 5. Test the application

Now that the application is deployed and running, it's time to test it. To do
this, you configure a DNS mapping on the machine where you are running your web
browser. This maps the "votingapp.local" DNS name to the public IP address of
the `interlock` node.

1. Configure the DNS name resolution on your local machine for browsing.

    - On Windows machines this is done by adding `votingapp.local <interlock-public-ip>` to the `C:\Windows\System32\Drivers\etc\hosts file`. Modifying this file requires administrator privileges. To open the file with administrator privileges, right-click `C:\Windows\System32\notepad.exe` and select `Run as administrator`. Once Notepad is open, click `file` > `open` and open the file and make the edit.
    - On OSX machines this is done by adding `votingapp.local <interlock-public-ip>` to `/private/etc/hosts`.
    - On most Linux machines this is done by adding `votingapp.local <interlock-public-ip>` to `/etc/hosts`.

    Be sure to replace `<interlock-public-ip>` with the public IP address of
    your `interlock` node. You can find the `interlock` node's Public IP by
    selecting your `interlock` EC2 Instance from within the AWS EC2 console.

2. Verify the mapping worked with a `ping` command from your local machine.

        ping votingapp.local
        Pinging votingapp.local [54.183.164.230] with 32 bytes of data:
        Reply from 54.183.164.230: bytes=32 time=164ms TTL=42
        Reply from 54.183.164.230: bytes=32 time=163ms TTL=42
        Reply from 54.183.164.230: bytes=32 time=169ms TTL=42

3. Point your web browser to [http://votingapp.local](http://votingapp.local)

    ![](../images/vote-app-test.jpg)

    Notice the text at the bottom of the web page. This shows which web
    container serviced the request. In the diagram above, this is `frontend02`.
    If you refresh your web browser you should see this change as the Interlock
    load balancer shares incoming requests across both web containers.

    To see more detailed load balancer data from the Interlock service, point your web browser to [http://stats:interlock@votingapp.local/haproxy?stats](http://stats:interlock@votingapp.local/haproxy?stats)

    ![](../images/proxy-test.jpg)

4. Cast your vote. It is recommended to choose "Dogs" ;-)

5. To see the results of the poll, you can point your web browser at the public IP of the `store` node

    ![](../images/poll-results.jpg)

## Next steps

Congratulations. You have successfully walked through manually deploying a
microservice-based application to a Swarm cluster. Of course, not every
deployment goes smoothly. Now that you've learned how to successfully deploy an
application at scale, you should learn [what to consider when troubleshooting
large applications running on a Swarm cluster](05-troubleshoot.md).
