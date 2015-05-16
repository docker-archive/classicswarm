package cli

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
		log.Fatalf("discovery required to join a cluster. See '%s join --help'.", c.App.Name)
	}

	hb, err := time.ParseDuration(c.String("heartbeat"))
	if err != nil {
		log.Fatalf("invalid --heartbeat: %v", err)
	}
	if hb < 1*time.Second {
		log.Fatal("--heartbeat should be at least one second")
	}
	d, err := discovery.New(dflag, hb)
	if err != nil {
		log.Fatal(err)
	}

	addr := c.String("addr")

	if !checkAddrFormat(addr) {
		log.Fatal("--addr should be of the form ip:port or hostname:port")
	}

	hbval := time.Duration(hb) * time.Second
	for {
		log.WithFields(log.Fields{"addr": addr, "discovery": dflag}).Infof("Registering on the discovery service every %s...", hbval)
		if err := d.Register(addr); err != nil {
			log.Error(err)
		}
		time.Sleep(hbval)
	}
}
