package cli

import (
	"fmt"
	"os"
	"path"
	"runtime/debug"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/experimental"
	"github.com/docker/swarm/version"
)

// Run the Swarm CLI.
func Run() {
	setProgramLimits()

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

		cli.BoolFlag{
			Name:  "experimental",
			Usage: "enable experimental features",
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

		experimental.ENABLED = c.Bool("experimental")

		return nil
	}

	app.Commands = commands

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setProgramLimits() {
	// Swarm runnable threads could be large when the number of nodes is large
	// or under request bursts. Most threads are occupied by network connections.
	// Increase max thread count from 10k default to 50k to accommodate it.
	const maxThreadCount int = 50 * 1000
	debug.SetMaxThreads(maxThreadCount)
}
