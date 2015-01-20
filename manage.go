package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/docker/swarm/state"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	log.Printf("event -> status: %q from: %q id: %q node: %q", e.Status, e.From, e.Id, e.Node.Name)
	return nil
}

// Load the TLS certificates/keys and, if verify is true, the CA.
func loadTlsConfig(ca, cert, key string, verify bool) (*tls.Config, error) {
	c, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("Couldn't load X509 key pair (%s, %s): %s. Key encrypted?",
			cert, key, err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{c},
		MinVersion:   tls.VersionTLS10,
	}

	if verify {
		certPool := x509.NewCertPool()
		file, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("Couldn't read CA certificate: %s", err)
		}
		certPool.AppendCertsFromPEM(file)
		config.RootCAs = certPool
	} else {
		// If --tlsverify is not supplied, disable CA validation.
		config.InsecureSkipVerify = true
	}

	return config, nil
}

func manage(c *cli.Context) {
	var (
		tlsConfig *tls.Config = nil
		err       error
	)

	// If either --tls or --tlsverify are specified, load the certificates.
	if c.Bool("tls") || c.Bool("tlsverify") {
		tlsConfig, err = loadTlsConfig(
			c.String("tlscacert"),
			c.String("tlscert"),
			c.String("tlskey"),
			c.Bool("tlsverify"))
		if err != nil {
			log.Fatal(err)
		}
	}

	store := state.NewStore(path.Join(c.String("rootdir"), "state"))
	if err := store.Initialize(); err != nil {
		log.Fatal(err)
	}

	cluster := cluster.NewCluster(store, tlsConfig, c.Float64("overcommit"))
	cluster.Events(&logHandler{})

	if c.String("discovery") == "" {
		log.Fatal("--discovery required to manage a cluster")
	}

	s, err := strategy.New(c.String("strategy"))
	if err != nil {
		log.Fatal(err)
	}

	// see https://github.com/codegangsta/cli/issues/160
	names := c.StringSlice("filter")
	if c.IsSet("filter") || c.IsSet("f") {
		names = names[3:]
	}
	fs, err := filter.New(names)
	if err != nil {
		log.Fatal(err)
	}

	// get the list of nodes from the discovery service
	go func() {
		d, err := discovery.New(c.String("discovery"), c.Int("heartbeat"))
		if err != nil {
			log.Fatal(err)
		}

		nodes, err := d.Fetch()
		if err != nil {
			log.Fatal(err)

		}
		cluster.UpdateNodes(nodes)

		go d.Watch(cluster.UpdateNodes)
	}()

	sched := scheduler.NewScheduler(
		cluster,
		s,
		fs,
	)

	// see https://github.com/codegangsta/cli/issues/160
	hosts := c.StringSlice("host")
	if c.IsSet("host") || c.IsSet("H") {
		hosts = hosts[1:]
	}
	log.Fatal(api.ListenAndServe(cluster, sched, hosts, c.App.Version, c.Bool("cors"), tlsConfig))
}
