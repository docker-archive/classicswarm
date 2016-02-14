# Dockerfile for swarm integration test environment.
# Use with run_in_docker.sh

FROM dockerswarm/dind:1.9.0

# Install dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends git file parallel \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install golang
ENV GO_VERSION 1.5.3
RUN curl -sSL https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

# Go dependencies
RUN go get github.com/tools/godep

# install bats
RUN cd /usr/local/src/ \
    && git clone https://github.com/sstephenson/bats.git \
    && cd bats \
    && ./install.sh /usr/local

RUN mkdir -p /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm/test/integration
ENV GOPATH /go/src/github.com/docker/swarm/Godeps/_workspace:$GOPATH

ENTRYPOINT ["/dind"]
