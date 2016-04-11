package cli

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

func maintenance(c *cli.Context) {
	if maintenanceNode == "" {
		log.Fatalf("need node to manage. See '%s maintenance --help'.", c.App.Name)
	}

	fmt.Printf("Setting maintenance on %s\n", maintenanceNode)
}
