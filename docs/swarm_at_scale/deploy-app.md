---
advisory: swarm-standalone
hide-from-sitemap: true
description: Try swarm at scale
keywords: docker, swarm, scale, voting, application, certificates
redirect_from:
- /swarm/swarm_at_scale/04-deploy-app/
title: Deploy the application
---

You've
[deployed the load balancer, the discovery backend, and a swarm cluster](deploy-infra.md)
so now you can build and deploy the voting application itself. You do this by
starting a number of "Dockerized applications" running in containers.

The diagram below shows the final application configuration including the overlay
container network, `voteapp`.

![Voteapp deployment overview](/swarm/images/final-result.png)

In this procedure you connect containers to this network. The `voteapp`
network is available to all Docker hosts using the Consul discovery backend.
Notice that the `interlock`, `nginx`, `consul`, and `swarm manager` containers
on are not part of the `voteapp` overlay container network.

## Task 1. Set up volume and network

This application relies on both an overlay container network and a container
volume. The Docker Engine provides these two features. Create them both
on the swarm manager instance.

1. Direct your local environment to the swarm manager host.

   ```bash
   $ eval $(docker-machine env manager)
   ```

    You can create the network on a cluster node and the network is visible on
    them all.

2. Create the `voteapp` container network.

   ```bash
   $ docker network create -d overlay voteapp
   ```

3. Switch to the db store.

   ```bash
   $ eval $(docker-machine env dbstore)
   ```

4. Verify you can see the new network from the dbstore node.

   ```bash
   $ docker network ls
   NETWORK ID          NAME                DRIVER
   e952814f610a        voteapp             overlay
   1f12c5e7bcc4        bridge              bridge
   3ca38e887cd8        none                null
   3da57c44586b        host                host
   ```

3. Create a container volume called `db-data`.

   ```bash
   $ docker volume create --name db-data
   ```

## Task 2. Start the containerized microservices

At this point, you are ready to start the component microservices that make up
the application. Some of the application's containers are launched from existing
images pulled directly from Docker Hub. Other containers are launched from
custom images you must build. The list below shows which containers use custom
images and which do not:

- Load balancer container: stock image (`ehazlett/interlock`)
- Redis containers: stock image (official `redis` image)
- Postgres (PostgreSQL) containers: stock image (official `postgres` image)
- Web containers: custom built image
- Worker containers: custom built image
- Results containers: custom built image

You can launch these containers from any host in the cluster using the commands
in this section. Each command includes a `-H `flag so that they execute against
the swarm manager.

The commands also all use the `-e` flag which is a Swarm constraint. The
constraint tells the manager to look for a node with a matching function label.
You set established the labels when you created the nodes. As you run each
command below, look for the value constraint.

1. Start a Postgres database container.

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -v db-data:/var/lib/postgresql/data \
   -e constraint:com.function==dbstore \
   --net="voteapp" \
   --name db postgres:9.4
   ```

6. Start the Redis container.

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -p 6379:6379 \
   -e constraint:com.function==dbstore \
   --net="voteapp" \
   --name redis redis
   ```

    The `redis` name is important so don't change it.

7. Start the worker application

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -e constraint:com.function==worker01 \
   --net="voteapp" \
   --net-alias=workers \
   --name worker01 docker/example-voting-app-worker
   ```

6. Start the results application.

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -p 80:80 \
   --label=interlock.hostname=results \
   --label=interlock.domain=myenterprise.example.com \
   -e constraint:com.function==dbstore \
   --net="voteapp" \
   --name results-app docker/example-voting-app-result
   ```

7. Start the voting application twice; once on each frontend node.

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -p 80:80 \
   --label=interlock.hostname=vote \
   --label=interlock.domain=myenterprise.example.com \
   -e constraint:com.function==frontend01 \
   --net="voteapp" \
   --name voting-app01 docker/example-voting-app-vote
   ```

    And again on the other frontend node.

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -p 80:80 \
   --label=interlock.hostname=vote \
   --label=interlock.domain=myenterprise.example.com \
   -e constraint:com.function==frontend02 \
   --net="voteapp" \
   --name voting-app02 docker/example-voting-app-vote
   ```


## Task 3. Check your work and update /etc/hosts

In this step, you check your work to make sure the Nginx configuration recorded
the containers correctly. Update your local system's `/etc/hosts` file to
allow you to take advantage of the loadbalancer.

1. Change to the `loadbalancer` node.

   ```bash
   $ eval $(docker-machine env loadbalancer)
   ```
2. Check your work by reviewing the configuration of nginx.

   ```html
   $ docker container exec interlock cat /etc/conf/nginx.conf
   ... output snipped ...

   upstream results.myenterprise.example.com {
       zone results.myenterprise.example.com_backend 64k;

       server 192.168.99.111:80;

   }
   server {
       listen 80;

       server_name results.myenterprise.example.com;

       location / {
           proxy_pass http://results.myenterprise.example.com;
       }
   }
   upstream vote.myenterprise.example.com {
       zone vote.myenterprise.example.com_backend 64k;

       server 192.168.99.109:80;
       server 192.168.99.108:80;

   }
   server {
       listen 80;

       server_name vote.myenterprise.example.com;

       location / {
           proxy_pass http://vote.myenterprise.example.com;
       }
   }

   include /etc/conf/conf.d/*.conf;
   }
   ```

    The `http://vote.myenterprise.example.com` site configuration should point to either
    frontend node. Requests to `http://results.myenterprise.example.com` go just to the
    single `dbstore` node where the `example-voting-app-result` is running.

3. On your local host, edit `/etc/hosts` file to add the resolution for both these
sites.

4. Save and close the `/etc/hosts` file.

5. Restart the `nginx` container.

    Manual restart is required because the current Interlock server is not forcing an
    Nginx configuration reload.

   ```bash
   $ docker restart nginx
   ```

## Task 4. Test the application

Now, you can test your application.

1. Open a browser and navigate to the `http://vote.myenterprise.example.com` site.

    You should see something similar to the following:

    ![Voting page](/swarm/images/vote-app-test.png)

2. Click on one of the two voting options.
3. Navigate to the `http://results.myenterprise.example.com` site to see the results.
4. Try changing your vote.

    Both sides change as you switch your vote.

    ![Voting and results page](/swarm/images/votes.gif)

## Extra Credit: Deployment with Docker Compose

Up to this point, you've deployed each application container individually. This
can be cumbersome especially because there are several different containers and
starting them is order dependent. For example, that database should be running
before the worker.

Docker Compose let's you define your microservice containers and their
dependencies in a Compose file. Then, you can use the Compose file to start all
the containers at once. This extra credit

1. Before you begin, stop all the containers you started.

    a. Set the host to the manager.

        $ DOCKER_HOST=$(docker-machine ip manager):3376

    b. List all the application containers on the swarm.

    c. Stop and remove each container.

2. Try to create Compose file on your own by reviewing the tasks in this tutorial.

    [The version 2 Compose file format](../../compose/compose-file/compose-file-v2.md)
    is the best to use. Translate each `docker run` command into a service
    in the `docker-compose.yml` file. For example,
    this command:

   ```none
   $ docker -H $(docker-machine ip manager):3376 run -t -d \
   -e constraint:com.function==worker01 \
   --net="voteapp" \
   --net-alias=workers \
   --name worker01 docker/example-voting-app-worker
   ```

    Becomes this in a Compose file.

   ```bash
   worker:
     image: docker/example-voting-app-worker
     networks:
       voteapp:
         aliases:
         - workers
   ```

    In general, Compose starts services in reverse order they appear in the file.
    So, if you want a service to start before all the others, make it the last
    service in the file. This application relies on a volume and a network,
    declare those at the bottom of the file.

3. Check your work against
   [this file](/swarm/swarm_at_scale/docker-compose.yml){: target="_blank" class="_"}.

4. When you are satisfied, save the `docker-compose.yml` file to your system.

5. Set `DOCKER_HOST` to the swarm manager.

   ```bash
   $ DOCKER_HOST=$(docker-machine ip manager):3376
   ```

6. In the same directory as your `docker-compose.yml` file, start the services.

   ```bash
   $ docker-compose up -d
   Creating network "scale_voteapp" with the default driver
   Creating volume "scale_db-data" with default driver
   Pulling db (postgres:9.4)...
   worker01: Pulling postgres:9.4... : downloaded
   dbstore: Pulling postgres:9.4... : downloaded
   frontend01: Pulling postgres:9.4... : downloaded
   frontend02: Pulling postgres:9.4... : downloaded
   Creating db
   Pulling redis (redis:latest)...
   dbstore: Pulling redis:latest... : downloaded
   frontend01: Pulling redis:latest... : downloaded
   frontend02: Pulling redis:latest... : downloaded
   worker01: Pulling redis:latest... : downloaded
   Creating redis
   Pulling worker (docker/example-voting-app-worker:latest)...
   dbstore: Pulling docker/example-voting-app-worker:latest... : downloaded
   frontend01: Pulling docker/example-voting-app-worker:latest... : downloaded
   frontend02: Pulling docker/example-voting-app-worker:latest... : downloaded
   worker01: Pulling docker/example-voting-app-worker:latest... : downloaded
   Creating scale_worker_1
   Pulling voting-app (docker/example-voting-app-vote:latest)...
   dbstore: Pulling docker/example-voting-app-vote:latest... : downloaded
   frontend01: Pulling docker/example-voting-app-vote:latest... : downloaded
   frontend02: Pulling docker/example-voting-app-vote:latest... : downloaded
   worker01: Pulling docker/example-voting-app-vote:latest... : downloaded
   Creating scale_voting-app_1
   Pulling result-app (docker/example-voting-app-result:latest)...
   dbstore: Pulling docker/example-voting-app-result:latest... : downloaded
   frontend01: Pulling docker/example-voting-app-result:latest... : downloaded
   frontend02: Pulling docker/example-voting-app-result:latest... : downloaded
   worker01: Pulling docker/example-voting-app-result:latest... : downloaded
   Creating scale_result-app_1
   ```

7. Use the `docker ps` command to see the containers on the swarm cluster.

   ```bash
   $ docker -H $(docker-machine ip manager):3376 ps
   CONTAINER ID        IMAGE                                  COMMAND                  CREATED             STATUS              PORTS                            NAMES
   b71555033caa        docker/example-voting-app-result   "node server.js"         6 seconds ago       Up 4 seconds        192.168.99.104:32774->80/tcp     frontend01/scale_result-app_1
   cf29ea21475d        docker/example-voting-app-worker       "/usr/lib/jvm/java-7-"   6 seconds ago       Up 4 seconds                                         worker01/scale_worker_1
   98414cd40ab9        redis                                  "/entrypoint.sh redis"   7 seconds ago       Up 5 seconds        192.168.99.105:32774->6379/tcp   frontend02/redis
   1f214acb77ae        postgres:9.4                           "/docker-entrypoint.s"   7 seconds ago       Up 5 seconds        5432/tcp                         frontend01/db
   1a4b8f7ce4a9        docker/example-voting-app-vote   "python app.py"          7 seconds ago       Up 5 seconds        192.168.99.107:32772->80/tcp     dbstore/scale_voting-app_1
   ```

    When you started the services manually, you had a
    `voting-app` instances running on two frontend servers. How many
    do you have now?

8. Scale your application up by adding some `voting-app` instances.

   ```bash
   $ docker-compose scale voting-app=3
   Creating and starting 2 ... done
   Creating and starting 3 ... done
   ```

    After you scale up, list the containers on the cluster again.

9. Change to the `loadbalancer` node.

   ```bash
   $ eval $(docker-machine env loadbalancer)
   ```

10. Restart the Nginx server.

    ```bash
    $ docker restart nginx
    ```
11. Check your work again by visiting the `http://vote.myenterprise.example.com` and
`http://results.myenterprise.example.com` again.

12. You can view the logs on an individual container.

    ```bash
    $ docker logs scale_voting-app_1
     * Running on http://0.0.0.0:80/ (Press CTRL+C to quit)
     * Restarting with stat
     * Debugger is active!
     * Debugger pin code: 285-809-660
    192.168.99.103 - - [11/Apr/2016 17:15:44] "GET / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:15:44] "GET /static/stylesheets/style.css HTTP/1.0" 304 -
    192.168.99.103 - - [11/Apr/2016 17:15:45] "GET /favicon.ico HTTP/1.0" 404 -
    192.168.99.103 - - [11/Apr/2016 17:22:24] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:37] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:39] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:40] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:41] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:43] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:44] "POST / HTTP/1.0" 200 -
    192.168.99.103 - - [11/Apr/2016 17:23:46] "POST / HTTP/1.0" 200 -
    ```

  This log shows the activity on one of the active voting application containers.

## Next steps

Congratulations. You have successfully walked through manually deploying a
microservice-based application to a swarm cluster. Of course, not every
deployment goes smoothly. Now that you've learned how to successfully deploy an
application at scale, you should learn what to consider when
[troubleshooting large applications running on a swarm cluster](troubleshoot.md).
