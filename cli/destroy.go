package cli

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery/token"
)

func destroy(c *cli.Context) {
	if len(c.Args()) != 1 {
		log.Fatalf("the `destroy` command takes one argument. See '%s destroy --help'.", c.App.Name)
	}

	discovery := &token.Discovery{}
	discovery.Initialize(c.Args()[0], 0, 0)
	authToken, err := discovery.DestroyCluster()
	if err != nil {
		log.Fatal(err)
	}

	if authToken != "" {
		username, password, err := getUsernameAndPassword(c.String("username"), c.String("password"))
		if err != nil {
			log.Fatal(err)
		}

		if username == "" || password == "" {
			log.Fatalf("the `destroy` command requires a DockerHub username and password. See '%s destroy --help'.", c.App.Name)
		}

		if err := discovery.DestroySecureCluster(authToken, username, password); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(c.Args()[0], "destroyed")
}
