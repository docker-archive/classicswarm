package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/docker/swarm/discovery"
	_ "github.com/docker/swarm/discovery/file"
	"github.com/docker/swarm/discovery/token"
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
	flDiscovery := cli.StringFlag{
		Name:   "discovery",
		Value:  "",
		Usage:  "token://<token>, file://path/to/file",
		EnvVar: "SWARM_DISCOVERY",
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
	flTls := cli.BoolFlag{
		Name:  "tls",
		Usage: "Use TLS; implied by --tlsverify=true",
	}
	flTlsCaCert := cli.StringFlag{
		Name:  "tlscacert",
		Usage: "Trust only remotes providing a certificate signed by the CA given here",
	}
	flTlsCert := cli.StringFlag{
		Name:  "tlscert",
		Usage: "Path to TLS certificate file",
	}
	flTlsKey := cli.StringFlag{
		Name:  "tlskey",
		Usage: "Path to TLS key file",
	}
	flTlsVerify := cli.BoolFlag{
		Name:  "tlsverify",
		Usage: "Use TLS and verify the remote",
	}

	app.Commands = []cli.Command{
		{
			Name:      "create",
			ShortName: "c",
			Usage:     "create a cluster",
			Action: func(c *cli.Context) {
				token, err := token.CreateCluster()
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
			Flags:     []cli.Flag{flDiscovery},
			Action: func(c *cli.Context) {
				if c.String("discovery") == "" {
					log.Fatal("--discovery required to list a cluster")
				}

				d, err := discovery.New(c.String("discovery"))
				if err != nil {
					log.Fatal(err)
				}

				nodes, err := d.Fetch()
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
			Flags: []cli.Flag{
				flDiscovery, flAddr, flHeartBeat,
				flTls, flTlsCaCert, flTlsCert, flTlsKey, flTlsVerify,
				flEnableCors},
			Action: manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     []cli.Flag{flDiscovery, flAddr, flHeartBeat},
			Action:    join,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
