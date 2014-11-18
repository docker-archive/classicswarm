package main

import (
	"log"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/libcluster/api"
	"github.com/docker/libcluster/discovery"
	"github.com/docker/libcluster/scheduler"
	"github.com/docker/libcluster/scheduler/filter"
	"github.com/docker/libcluster/scheduler/strategy"
	"github.com/docker/libcluster/swarm"
)

func manage(c *cli.Context) {

	refresh := func(cluster *swarm.Cluster, nodes []string) error {
		for _, addr := range nodes {
			if cluster.Node(addr) == nil {
				n := swarm.NewNode(addr, addr)
				if err := n.Connect(nil); err != nil {
					return err
				}
				if err := cluster.AddNode(n); err != nil {
					return err
				}
			}
		}
		return nil
	}

	cluster := swarm.NewCluster()
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
