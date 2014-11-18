package main

import (
	"log"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
)

func manage(c *cli.Context) {

	refresh := func(c *cluster.Cluster, nodes []string) error {
		for _, addr := range nodes {
			if c.Node(addr) == nil {
				n := cluster.NewNode(addr, addr)
				if err := n.Connect(nil); err != nil {
					return err
				}
				if err := c.AddNode(n); err != nil {
					return err
				}
			}
		}
		return nil
	}

	cluster := cluster.NewCluster()
	cluster.Events(&logHandler{})

	if c.String("token") != "" {
		nodes, err := discovery.FetchSlaves(c.String("token"))
		if err != nil {
			log.Fatal(err)

		}
		if err := refresh(cluster, nodes); err != nil {
			log.Fatal(err)
		}
		go func() {
			for {
				time.Sleep(25 * time.Second)
				nodes, err = discovery.FetchSlaves(c.String("token"))
				if err == nil {
					refresh(cluster, nodes)
				}
			}
		}()
	} else {
		if err := refresh(cluster, c.Args()); err != nil {
			log.Fatal(err)
		}
	}

	s := scheduler.NewScheduler(cluster, &strategy.BinPackingPlacementStrategy{}, []filter.Filter{})

	log.Fatal(api.ListenAndServe(cluster, s, c.String("addr")))
}
