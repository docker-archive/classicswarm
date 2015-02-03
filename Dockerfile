FROM golang:1.3

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

ENV GOPATH /go/src/github.com/docker/swarm/Godeps/_workspace:$GOPATH
RUN CGO_ENABLED=0 go install -v -a -tags netgo -ldflags "-w -X github.com/docker/swarm/api.GITCOMMIT `git rev-parse --short HEAD`"

EXPOSE 2375
VOLUME $HOME/.swarm

ENTRYPOINT ["swarm"]
CMD ["--help"]
