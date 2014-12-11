package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

func join(c *cli.Context) {

	if c.String("discovery") == "" {
		log.Fatal("--discovery required to join a cluster")
	}

	d, err := discovery.New(c.String("discovery"), c.Int("heartbeat"))
	if err != nil {
		log.Fatal(err)
	}

	if err := d.Register(c.String("addr")); err != nil {
		log.Fatal(err)
	}

	hb := c.Duration("heartbeat")
	for {
		time.Sleep(hb * time.Second)
		if err := d.Register(c.String("addr")); err != nil {
			log.Error(err)
		}
	}
}
