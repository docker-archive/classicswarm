#!/bin/sh

curl -H "Content-Type: application/json" --data @redis.json http://mesos-master:8080/v2/apps