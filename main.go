package main

import (
	_ "github.com/docker/swarm/discovery/file"
	_ "github.com/docker/swarm/discovery/kv"
	_ "github.com/docker/swarm/discovery/nodes"
	_ "github.com/docker/swarm/discovery/token"

	"github.com/docker/swarm/cli"
)

func main() {
	cli.Run()
}
