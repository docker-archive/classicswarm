FROM    debian:jessie

RUN     apt-get update && \
        apt-get install -y --no-install-recommends ca-certificates

COPY    dist/bin/swarm /swarm
ENV     SWARM_HOST :2375
EXPOSE  2375
VOLUME  /.swarm

ENTRYPOINT ["/swarm"]
CMD ["--help"]
