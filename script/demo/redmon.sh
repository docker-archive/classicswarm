#!/bin/sh

# start a redis server
docker run -d -P --name redis redis

# start redmon and point it to the redis container's IP:port
docker run -d -P --name redmon -e affinity:container!=redis vieux/redmon -r redis://`docker port redis 6379`

echo "http://`docker port redmon 4567`"
