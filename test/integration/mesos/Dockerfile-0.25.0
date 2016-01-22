FROM debian:8

MAINTAINER Victor Vieux <vieux@docker.com>

RUN apt-get update && apt-get install wget python -y
RUN wget http://downloads.mesosphere.io/master/debian/8/mesos_0.25.0-0.2.70.debian81_amd64.deb -O /tmp/mesos.deb
RUN dpkg -i /tmp/mesos.deb || true
RUN apt-get install -f -y

USER daemon

