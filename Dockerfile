FROM golang:1.3

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

ENV GOPATH $GOPATH:/go/src/github.com/docker/swarm/Godeps/_workspace
RUN go install -v

EXPOSE 2375

ENTRYPOINT ["swarm"]
CMD ["--help"]
