# Dockerfile for swarm integration test environment.
# Use with run_in_docker.sh

FROM dockerswarm/dind:1.6.0

# Install dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        file \
        git \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install golang
ENV GO_VERSION 1.4.2
RUN curl -sSL https://golang.org/dl/go${GO_VERSION}.src.tar.gz | tar -v -C /usr/local -xz \
    && mkdir -p /go/bin
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go
RUN cd /usr/local/go/src && ./make.bash --no-clean 2>&1

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
