package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

func list(c *cli.Context) {
	dflag := getDiscovery(c)
	if dflag == "" {
		log.Fatalf("discovery required to list a cluster. See '%s list --help'.", c.App.Name)
	}
	timeout, err := time.ParseDuration(c.String("timeout"))
	if err != nil {
		log.Fatalf("invalid --timeout: %v", err)
	}

	d, err := discovery.New(dflag, timeout, 0)
	if err != nil {
		log.Fatal(err)
	}

	ch, errCh := d.Watch(nil)
	select {
	case entries := <-ch:
		for _, entry := range entries {
			fmt.Println(entry)
		}
	case err := <-errCh:
		log.Fatal(err)
	case <-time.After(timeout):
		log.Fatal("Timed out")
	}
}
