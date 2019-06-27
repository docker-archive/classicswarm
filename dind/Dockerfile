# Dockerfile for building Docker
FROM debian:jessie

# compile and runtime deps
# https://github.com/docker/docker/blob/master/project/PACKAGERS.md#runtime-dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        iptables \
        procps \
        e2fsprogs \
        xz-utils \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV VERSION dev
RUN curl -L -o /usr/local/bin/docker https://master.dockerproject.org/linux/x86_64/docker \
    && chmod +x /usr/local/bin/docker
RUN curl -L -o /usr/local/bin/dockerd https://master.dockerproject.org/linux/x86_64/dockerd \
    && chmod +x /usr/local/bin/dockerd
RUN curl -L -o /usr/local/bin/runc https://master.dockerproject.org/linux/x86_64/docker-runc \
    && chmod +x /usr/local/bin/runc
RUN curl -L -o /usr/local/bin/containerd https://master.dockerproject.org/linux/x86_64/docker-containerd \
    && chmod +x /usr/local/bin/containerd
RUN curl -L -o /usr/local/bin/containerd-shim https://master.dockerproject.org/linux/x86_64/docker-containerd-shim \
    && chmod +x /usr/local/bin/containerd-shim
RUN curl -L -o /usr/local/bin/docker-proxy https://master.dockerproject.org/linux/x86_64/docker-proxy \
    && chmod +x /usr/local/bin/docker-proxy

RUN curl -L -o /dind https://raw.githubusercontent.com/docker/docker/master/hack/dind \
    && chmod +x /dind

VOLUME /var/lib/docker

ENTRYPOINT ["/dind"]
