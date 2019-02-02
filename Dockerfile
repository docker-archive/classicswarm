FROM golang:1.7.1 AS builder

ARG GOOS
ARG GOARCH

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

RUN GOARCH=$GOARCH GOOS=$GOOS CGO_ENABLED=0 go build -a -installsuffix cgo -o swarm



FROM alpine
MAINTAINER QinWuquan <jamesqin@vip.qq.com>
COPY --from=builder /go/src/github.com/docker/swarm/swarm /bin/swarm
ENV SWARM_HOST :2375
EXPOSE 2375
ENTRYPOINT ["/bin/swarm"]
CMD ["--help"]
