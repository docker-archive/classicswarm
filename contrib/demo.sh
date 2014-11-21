#!/bin/sh

# clean up cluster
docker rm -f `docker ps -aq`

# step1
docker info
docker run -d -P --name redis redis
docker ps
docker port redis
docker inspect redis
docker logs redis
docker rm -f redis

# step2

docker run -d -m 10g redis

docker run -d -m 1g redis
docker ps
docker run -d -m 1g redis
docker ps
id=`docker run -d -m 1g redis`
docker ps
docker run -d -m 2g redis
docker ps
docker rm -f $id
docker run -d -m 2g redis

# clean up cluster
docker rm -f `docker ps -aq`

# step3

docker run -d -p 80:80 nginx
docker run -d -p 80:80 nginx
id=`docker run -d -p 80:80 nginx`
docker ps

docker rm -f $id
docker run -d -p 80:80 nginx

# clean up cluster
docker rm -f `docker ps -aq`

docker run -d -e constraint:operatingsystem=fedora redis
docker ps

docker run -d -e constraint:storagedriver=devicemapper redis
docker ps

docker run -d -e constraint:storagedriver=aufs redis
docker ps

docker run -d -e constraint:node=fedora-1 redis
docker ps

# clean up cluster
docker rm -f `docker ps -aq`
