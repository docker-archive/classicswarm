package cli

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

func checkAddrFormat(addr string) bool {
	m, _ := regexp.MatchString("^[0-9a-zA-Z._-]+:[0-9]{1,5}$", addr)
	return m
}

func checkAddrValidity(addr string) bool {
	newip := strings.Split(addr, ":")
	// Handle loopback interface as a special case
	if strings.EqualFold(newip[0], "localhost") {
		return true
	}
	// Compare host's network interface addresses with requested address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	for _, val := range addrs {
		ipval := strings.Split(val.String(), "/")
		if strings.EqualFold(newip[0], ipval[0]) {
			return true
		}
	}
	return false
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

	if !checkAddrValidity(addr) {
		log.Fatalf("requested address %s not associated with any network interface", addr)
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
