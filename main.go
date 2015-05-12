package main

import (
	_ "github.com/docker/swarm/discovery/consul"
	_ "github.com/docker/swarm/discovery/digitalocean"
	_ "github.com/docker/swarm/discovery/etcd"
	_ "github.com/docker/swarm/discovery/file"
	_ "github.com/docker/swarm/discovery/nodes"
	_ "github.com/docker/swarm/discovery/token"
	_ "github.com/docker/swarm/discovery/zookeeper"

	"github.com/docker/swarm/cli"
)

func main() {
	cli.Run()
}
