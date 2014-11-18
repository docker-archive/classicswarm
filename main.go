package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libcluster/api"
	"github.com/docker/libcluster/discovery"
	"github.com/docker/libcluster/scheduler"
	"github.com/docker/libcluster/scheduler/filter"
	"github.com/docker/libcluster/scheduler/strategy"
	"github.com/docker/libcluster/swarm"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *swarm.Event) error {
	log.Printf("event -> type: %q time: %q image: %q container: %q", e.Type, e.Time.Format(time.RubyDate), e.Container.Image, e.Container.Id)
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "swarm"
	app.Usage = "docker clustering"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	clusterFlags := []cli.Flag{
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
	}

	app.Commands = []cli.Command{
		{
			Name:      "create",
			ShortName: "c",
			Usage:     "create a cluster",

			Action: func(c *cli.Context) {
				token, err := discovery.CreateCluster()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(token)
			},
		},
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "manage a docker cluster",
			Flags:     clusterFlags,

			Action: func(c *cli.Context) {

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
			},
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     clusterFlags,

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

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
