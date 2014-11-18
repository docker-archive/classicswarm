package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libcluster/discovery"
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
			Action:    manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     clusterFlags,
			Action:    join,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
