package main

import (
	"regexp"
	"strconv"
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
		log.Fatalf("discovery required to join a cluster. See '%s join --help'.", c.App.Name)
	}

	hb, err := strconv.ParseUint(c.String("heartbeat"), 0, 32)
	if hb < 1 || err != nil {
		log.Fatal("--heartbeat should be an unsigned integer and greater than 0")
	}

	d, err := discovery.New(dflag, hb)
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

	hbval := time.Duration(hb)
	for {
		log.WithFields(log.Fields{"addr": addr, "discovery": dflag}).Infof("Registering on the discovery service every %d seconds...", hbval)
		time.Sleep(hbval * time.Second)
		if err := d.Register(addr); err != nil {
			log.Error(err)
		}
	}
}
