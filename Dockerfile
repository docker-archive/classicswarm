FROM golang:1.5.4-alpine

ARG GOOS

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

ENV GO15VENDOREXPERIMENT=1

RUN set -ex \
	&& apk add --no-cache --virtual .build-deps \
	git \
	&& GOOS=$GOOS CGO_ENABLED=0 go install -v -a -tags netgo -installsuffix netgo -ldflags "-w -X github.com/docker/swarm/version.GITCOMMIT `git rev-parse --short HEAD` -X github.com/docker/swarm/version.BUILDTIME \"`date -u`\""  \
	&& apk del .build-deps

ENV SWARM_HOST :2375
EXPOSE 2375

VOLUME $HOME/.swarm

ENTRYPOINT ["swarm"]
CMD ["--help"]
