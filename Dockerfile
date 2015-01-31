FROM golang:1.3

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

ENV GOPATH $GOPATH:/go/src/github.com/docker/swarm/Godeps/_workspace
RUN CGO_ENABLED=0 go install -v -a -tags netgo -ldflags '-w'

EXPOSE 2375
VOLUME $HOME/.swarm

ENTRYPOINT ["swarm"]
CMD ["--help"]
