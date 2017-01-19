FROM golang:1.7.1-alpine

ARG GOOS
ARG GOARCH

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

RUN set -ex \
	&& apk add --no-cache --virtual .build-deps \
	git \
	&& GOARCH=$GOARCH GOOS=$GOOS CGO_ENABLED=0 go install -v -a -tags netgo -installsuffix netgo -ldflags "-w -X github.com/docker/swarm/version.GITCOMMIT=$(git rev-parse --short HEAD) -X github.com/docker/swarm/version.BUILDTIME=$(date -u +%FT%T%z)"  \
	&& apk del .build-deps

ENV SWARM_HOST :2375
EXPOSE 2375

VOLUME $HOME/.swarm

ENTRYPOINT ["swarm"]
CMD ["--help"]
