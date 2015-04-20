package cli

import (
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/discovery/token"
	"github.com/docker/swarm/version"
)

// Run the Swarm CLI.
func Run() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a Docker-native clustering system"
	app.Version = version.VERSION + " (" + version.GITCOMMIT + ")"

	app.Author = ""
	app.Email = ""

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},

		cli.StringFlag{
			Name:  "log-level, l",
			Value: "info",
			Usage: fmt.Sprintf("Log level (options: debug, info, warn, error, fatal, panic)"),
		},
	}

	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.SetLevel(level)

		// If a log level wasn't specified and we are running in debug mode,
		// enforce log-level=debug.
		if !c.IsSet("log-level") && !c.IsSet("l") && c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      "create",
			ShortName: "c",
			Usage:     "create a cluster",
			Action: func(c *cli.Context) {
				discovery := &token.Discovery{}
				discovery.Initialize("", 0)
				token, err := discovery.CreateCluster()
				if len(c.Args()) != 0 {
					log.Fatalf("the `create` command takes no arguments. See '%s create --help'.", c.App.Name)
				}
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
			Action: func(c *cli.Context) {
				dflag := getDiscovery(c)
				if dflag == "" {
					log.Fatalf("discovery required to list a cluster. See '%s list --help'.", c.App.Name)
				}

				d, err := discovery.New(dflag, 0)
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
				flStore,
				flStrategy, flFilter,
				flHosts, flHeartBeat, flOverCommit,
				flTLS, flTLSCaCert, flTLSCert, flTLSKey, flTLSVerify,
				flEnableCors},
			Action: manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     []cli.Flag{flAddr, flHeartBeat},
			Action:    join,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
