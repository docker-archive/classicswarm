package cli

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/experimental"
	"github.com/docker/swarm/version"
	"os"
	"path"
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

		cli.BoolFlag{
			Name:  "experimental",
			Usage: "enable experimental features",
		},
		cli.StringFlag{
			Name:  "log-file",
			Usage: "Logging the output to a file instead of stdout",
		},
	}
	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		if c.IsSet("log-file") {
			logDir := c.String("log-file")
			file, err := os.OpenFile(logDir, os.O_RDWR, 0777)
			if err != nil {
				//log.Warn("Error creating log file: falling back to default method")
				log.Fatalf(err.Error())
			} else {
				log.SetOutput(file)
			}
		}
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
