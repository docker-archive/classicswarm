package main

import (
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	log.Printf("event -> status: %q from: %q id: %q node: %q", e.Status, e.From, e.Id, e.NodeName)
	return nil
}

func manage(c *cli.Context) {

	refresh := func(c *cluster.Cluster, nodes []string) {
		for _, addr := range nodes {
			go func(addr string) {
				if !strings.Contains(addr, "://") {
					addr = "http://" + addr
				}
				if c.Node(addr) == nil {
					n := cluster.NewNode(addr)
					if err := n.Connect(nil); err != nil {
						log.Error(err)
						return
					}
					if err := c.AddNode(n); err != nil {
						log.Error(err)
						return
					}
				}
			}(addr)
		}
	}

	cluster := cluster.NewCluster()
	cluster.Events(&logHandler{})

	go func() {
		if c.String("token") != "" {
			nodes, err := discovery.FetchSlaves(c.String("token"))
			if err != nil {
				log.Fatal(err)

			}
			refresh(cluster, nodes)

			hb := time.Duration(c.Int("heartbeat"))
			go func() {
				for {
					time.Sleep(hb * time.Second)
					nodes, err = discovery.FetchSlaves(c.String("token"))
					if err == nil {
						refresh(cluster, nodes)
					}
				}
			}()
		} else {
			refresh(cluster, c.Args())
		}
	}()

	s := scheduler.NewScheduler(
		cluster,
		&strategy.BinPackingPlacementStrategy{OvercommitRatio: 0.05},
		[]filter.Filter{
			&filter.HealthFilter{},
			&filter.LabelFilter{},
			&filter.PortFilter{},
		},
	)

	log.Fatal(api.ListenAndServe(cluster, s, c.String("addr"), c.App.Version))
}
