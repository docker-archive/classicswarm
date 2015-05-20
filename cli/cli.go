package cli

import (
	"fmt"
	"os"
	"path"
	"time"

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
	app.Usage = "A Docker-native clustering system"
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
			Usage:     "Create a cluster",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 0 {
					log.Fatalf("the `create` command takes no arguments. See '%s create --help'.", c.App.Name)
				}
				discovery := &token.Discovery{}
				discovery.Initialize("", 0, 0)
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
			Usage:     "List nodes in a cluster",
			Flags:     []cli.Flag{flTimeout},
			Action: func(c *cli.Context) {
				dflag := getDiscovery(c)
				if dflag == "" {
					log.Fatalf("discovery required to list a cluster. See '%s list --help'.", c.App.Name)
				}
				timeout, err := time.ParseDuration(c.String("timeout"))
				if err != nil {
					log.Fatalf("invalid --timeout: %v", err)
				}

				d, err := discovery.New(dflag, timeout, 0)
				if err != nil {
					log.Fatal(err)
				}

				ch, errCh := d.Watch(nil)
				select {
				case entries := <-ch:
					for _, entry := range entries {
						fmt.Println(entry)
					}
				case err := <-errCh:
					log.Fatal(err)
				case <-time.After(timeout):
					log.Fatal("Timed out")
				}
			},
		},
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "Manage a docker cluster",
			Flags: []cli.Flag{
				flStore,
				flStrategy, flFilter,
				flHosts,
				flTLS, flTLSCaCert, flTLSCert, flTLSKey, flTLSVerify,
				flEnableCors,
				flCluster, flClusterOpt},
			Action: manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     []cli.Flag{flAddr, flHeartBeat, flTTL},
			Action:    join,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
