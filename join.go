package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

func join(c *cli.Context) {

	if c.String("token") == "" {
		log.Fatal("--token required to join a cluster")
	}

	if err := discovery.RegisterSlave(c.String("addr"), c.String("token")); err != nil {
		log.Fatal(err)
	}

	hb := time.Duration(c.Int("heartbeat"))
	for {
		time.Sleep(hb * time.Second)
		if err := discovery.RegisterSlave(c.String("addr"), c.String("token")); err != nil {
			log.Error(err)
		}
	}
}
