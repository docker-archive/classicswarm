package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/docker/swarm/discovery/token"
	"github.com/urfave/cli"
)

const tokenDeprecationErr = "Token based discovery is now deprecated and might be removed in the future.\nIt will be replaced by a default discovery backed by Docker Swarm Mode.\nOther mechanisms such as consul and etcd will continue to work as expected.\n"

func create(c *cli.Context) {
	if len(c.Args()) != 0 {
		log.Fatalf("the `create` command takes no arguments. See '%s create --help'.", c.App.Name)
	}
	discovery := &token.Discovery{}
	discovery.Initialize("", 0, 0, nil)
	token, err := discovery.CreateCluster()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, tokenDeprecationErr)
	fmt.Println(token)
}
