package main

import (
	"regexp"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

func checkAddrFormat(addr string) bool {
	m, _ := regexp.MatchString("^[0-9a-zA-Z._-]+:[0-9]{1,5}$", addr)
	return m
}

func join(c *cli.Context) {
	dflag := getDiscovery(c)
	if dflag == "" {
		log.Fatal("discovery required to join a cluster. See 'swarm join --help'.")
	}

	d, err := discovery.New(dflag, c.Int("heartbeat"))
	if err != nil {
		log.Fatal(err)
	}

	addr := c.String("addr")

	if !checkAddrFormat(addr) {
		log.Fatal("--addr should be of the form ip:port or hostname:port")
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
