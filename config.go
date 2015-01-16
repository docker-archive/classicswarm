package main

import (
	"crypto/tls"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
)

func config(c *cli.Context) {
	var (
		tlsConfig *tls.Config = nil
		err       error
	)

	// If either --tls or --tlsverify are specified, load the certificates.
	if c.Bool("tls") || c.Bool("tlsverify") {
		tlsConfig, err = loadTlsConfig(
			c.String("tlscacert"),
			c.String("tlscert"),
			c.String("tlskey"),
			c.Bool("tlsverify"))
		if err != nil {
			log.Fatal(err)
		}
	}

	if !c.IsSet("discovery") {
		log.Fatal("--discovery required to get config of a node")
	}

	if len(c.Args()) != 1 {
		log.Fatal("an argument is required to get config of a node")
	}
	// get the list of nodes from the discovery service
	d, err := discovery.New(c.String("discovery"), 0)
	if err != nil {
		log.Fatal(err)
	}

	nodes, err := d.Fetch()
	if err != nil {
		log.Fatal(err)

	}

	connectedNodes := make(chan *cluster.Node, len(nodes))
	for _, addr := range nodes {
		go func(node *discovery.Node) {
			n := cluster.NewNode(node.String(), 0)
			if err := n.Connect(tlsConfig); err != nil {
				log.Error(err)
			}
			connectedNodes <- n
		}(addr)
	}

	for n := range connectedNodes {
		if n.Name == c.Args().First() || n.ID == c.Args().First() {
			fmt.Println(n.Addr)
			return
		}
	}

	log.Fatalf("Node %q not found", c.Args().First())
}
