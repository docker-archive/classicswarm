package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
)

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

	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	// flags
	flToken := cli.StringFlag{
		Name:   "token",
		Value:  "",
		Usage:  "cluster token",
		EnvVar: "SWARM_TOKEN",
	}
	flAddr := cli.StringFlag{
		Name:   "addr",
		Value:  "127.0.0.1:4243",
		Usage:  "ip to advertise",
		EnvVar: "SWARM_ADDR",
	}
	flHeartBeat := cli.IntFlag{
		Name:  "heartbeat, hb",
		Value: 25,
		Usage: "time in second between each heartbeat",
	}
	flEnableCors := cli.BoolFlag{
		Name:  "api-enable-cors, cors",
		Usage: "enable CORS headers in the remote API",
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
			Name:      "list",
			ShortName: "l",
			Usage:     "list nodes in a cluster",
			Flags:     []cli.Flag{flToken},
			Action: func(c *cli.Context) {
				if c.String("token") == "" {
					log.Fatal("--token required to list a cluster")
				}

				nodes, err := discovery.FetchSlaves(c.String("token"))
				if err != nil {
					log.Fatal(err)
				}
				for _, node := range nodes {
					fmt.Println(node)
				}
			},
		},
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "manage a docker cluster",
			Flags:     []cli.Flag{flToken, flAddr, flHeartBeat, flEnableCors},
			Action:    manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     []cli.Flag{flToken, flAddr, flHeartBeat},
			Action:    join,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
