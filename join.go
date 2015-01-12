package main

import (
	"regexp"
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

	addr := c.String("addr")
	if m, _ := regexp.MatchString("^[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}:[0-9]{1,5}$", addr); !m {
		log.Fatal("--addr should be of the form ip:port")
	}

	if err := d.Register(addr); err != nil {
		log.Fatal(err)
	}

	hb := time.Duration(c.Int("heartbeat"))
	for {
		time.Sleep(hb * time.Second)
		if err := d.Register(c.String("addr")); err != nil {
			log.Error(err)
		}
	}
}
