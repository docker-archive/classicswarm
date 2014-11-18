package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libcluster"
	"github.com/docker/libcluster/api"
	"github.com/docker/libcluster/discovery"
	"github.com/docker/libcluster/scheduler"
	"github.com/docker/libcluster/scheduler/filter"
	"github.com/docker/libcluster/scheduler/strategy"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *libcluster.Event) error {
	log.Printf("event -> type: %q time: %q image: %q container: %q", e.Type, e.Time.Format(time.RubyDate), e.Container.Image, e.Container.Id)
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "swarm"
	app.Usage = "docker clustering"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "token",
			Value:  "",
			Usage:  "cluster token",
			EnvVar: "SWARM_TOKEN",
		},
		cli.StringFlag{
			Name:   "addr",
			Value:  "127.0.0.1:4243",
			Usage:  "ip to advertise",
			EnvVar: "SWARM_ADDR",
		},
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},
	}

	debug := func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "manage a docker cluster",
			Before:    debug,
			Action: func(c *cli.Context) {

				refresh := func(cluster *libcluster.Cluster, nodes []string) {
					for _, addr := range nodes {
						if cluster.Node(addr) == nil {
							n := libcluster.NewNode(addr, addr)
							if err := n.Connect(nil); err != nil {
								log.Fatal(err)
							}
							if err := cluster.AddNode(n); err != nil {
								log.Fatal(err)
							}
						}
					}
				}

				cluster := libcluster.NewCluster()
				cluster.Events(&logHandler{})

				if c.String("token") != "" {
					nodes, err := discovery.FetchSlaves(c.String("token"))
					if err != nil {
						log.Fatal(err)

					}
					refresh(cluster, nodes)
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
					refresh(cluster, c.Args()[1:])
				}

				s := scheduler.NewScheduler(cluster, &strategy.BinPackingPlacementStrategy{}, []filter.Filter{})

				log.Fatal(api.ListenAndServe(cluster, s, c.String("addr")))
			},
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Before:    debug,
			Action: func(c *cli.Context) {

				if c.String("token") == "" {
					log.Fatal("--token required to join a cluster")
				}

				if err := discovery.RegisterSlave(c.String("addr"), c.String("token")); err != nil {
					log.Fatal(err)
				}

				// heartbeat every 25 seconds
				go func() {
					for {
						time.Sleep(25 * time.Second)
						if err := discovery.RegisterSlave(c.String("addr"), c.String("token")); err != nil {
							log.Error(err)
						}
					}
				}()
			},
		},
	}

	app.Run(os.Args)
}
