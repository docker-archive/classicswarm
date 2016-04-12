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
	/*
		Goal: set maintenance flag for a node

		Look at list for inspiration, one prerequisite is that we find them, list does this. Can we update config to any node
		in a cluster?

		TODO: find where our node struct lives, and update it
		Q: should we set maintenance *on* the swarm node, and will/should discovery/replication propagate it?
		Q: *can* we modify properties on nodes? How is this done for healthyness?
		P: we have added the maintenance bool to the scheduler, how do access/modify it?
	*/
}
