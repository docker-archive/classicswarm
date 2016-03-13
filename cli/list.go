package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/discovery"
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
	if timeout <= time.Duration(0)*time.Second {
		log.Fatalf("--timeout should be a positive number")
	}

	d, err := discovery.New(dflag, timeout, 0, getDiscoveryOpt(c))
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
