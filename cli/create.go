package cli

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/discovery/token"
)

func create(c *cli.Context) {
	var (
		username = c.String("username")
		password = c.String("password")
		err      error
	)

	if username == "" && password == "" {
		fmt.Printf("Warning: Anonymous cluster created. Anonymous clusters can be destroyed by anyone with knowledge of the token. Please provide a `--username` in order to create an authenticated cluster.")
	} else {
		username, password, err = getUsernameAndPassword(username, password)
		if err != nil {
			log.Fatal(err)
		}

		if username == "" || password == "" {
			log.Fatalf("authenticated cluster creation require both username and password")
		}
	}
	discovery := &token.Discovery{}
	discovery.Initialize("", 0, 0)
	token, err := discovery.CreateCluster(username, password)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
}
