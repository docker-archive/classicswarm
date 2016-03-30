package cli

import (
	"math/rand"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/discovery"
)

func checkAddrFormat(addr string) bool {
	// validate addr is in host:port form. Use net function to handle both IPv4/IPv6 cases.
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	portNum, err := strconv.Atoi(port)
	return err == nil && portNum > 0 && portNum <= 65535
}

func join(c *cli.Context) {
	dflag := getDiscovery(c)
	if dflag == "" {
		log.Fatalf("discovery required to join a cluster. See '%s join --help'.", c.App.Name)
	}

	addr := c.String("advertise")
	if addr == "" {
		log.Fatal("missing mandatory --advertise flag")
	}
	if !checkAddrFormat(addr) {
		log.Fatal("--advertise should be of the form ip:port or hostname:port")
	}

	joinDelay, err := time.ParseDuration(c.String("delay"))
	if err != nil {
		log.Fatalf("invalid --delay: %v", err)
	}
	if joinDelay < time.Duration(0)*time.Second {
		log.Fatalf("--delay should not be a negative number")
	}

	hb, err := time.ParseDuration(c.String("heartbeat"))
	if err != nil {
		log.Fatalf("invalid --heartbeat: %v", err)
	}
	if hb < 1*time.Second {
		log.Fatal("--heartbeat should be at least one second")
	}
	ttl, err := time.ParseDuration(c.String("ttl"))
	if err != nil {
		log.Fatalf("invalid --ttl: %v", err)
	}
	if ttl <= hb {
		log.Fatal("--ttl must be strictly superior to the heartbeat value")
	}

	d, err := discovery.New(dflag, hb, ttl, getDiscoveryOpt(c))
	if err != nil {
		log.Fatal(err)
	}

	// if joinDelay is 0, no delay will be executed
	// if joinDelay is larger than 0,
	// add a random delay between 0s and joinDelay at start to avoid synchronized registration
	if joinDelay > 0 {
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		delay := time.Duration(r.Int63n(int64(joinDelay)))
		log.Infof("Add a random delay %s to avoid synchronized registration", delay)
		time.Sleep(delay)
	}

	for {
		log.WithFields(log.Fields{"addr": addr, "discovery": dflag}).Infof("Registering on the discovery service every %s...", hb)
		if err := d.Register(addr); err != nil {
			log.Error(err)
		}
		time.Sleep(hb)
	}
}
