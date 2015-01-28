FROM golang:1.3

ADD . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

RUN go get github.com/tools/godep
RUN godep go install

EXPOSE 2375
ENTRYPOINT ["swarm"]
