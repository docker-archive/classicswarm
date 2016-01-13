package main

import (
	_ "github.com/docker/docker/pkg/discovery/file"
	_ "github.com/docker/docker/pkg/discovery/kv"
	_ "github.com/docker/docker/pkg/discovery/nodes"
	_ "github.com/docker/swarm/discovery/token"

	"github.com/docker/swarm/cli"
)

func main() {
	cli.Run()
}
