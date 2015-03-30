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
	"github.com/docker/swarm/cluster/swarm"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/docker/swarm/state"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	id := e.Id
	// Trim IDs to 12 chars.
	if len(id) > 12 {
		id = id[:12]
	}
	log.WithFields(log.Fields{"node": e.Node.Name, "id": id, "from": e.From, "status": e.Status}).Debug("Event received")
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
		config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientCAs = certPool
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
		if !c.IsSet("tlscert") || !c.IsSet("tlskey") {
			log.Fatal("--tlscert and --tlskey must be provided when using --tls")
		}
		if c.Bool("tlsverify") && !c.IsSet("tlscacert") {
			log.Fatal("--tlscacert must be provided when using --tlsverify")
		}
		tlsConfig, err = loadTlsConfig(
			c.String("tlscacert"),
			c.String("tlscert"),
			c.String("tlskey"),
			c.Bool("tlsverify"))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Otherwise, if neither --tls nor --tlsverify are specified, abort if
		// the other flags are passed as they will be ignored.
		if c.IsSet("tlscert") || c.IsSet("tlskey") || c.IsSet("tlscacert") {
			log.Fatal("--tlscert, --tlskey and --tlscacert require the use of either --tls or --tlsverify")
		}
	}

	store := state.NewStore(path.Join(c.String("rootdir"), "state"))
	if err := store.Initialize(); err != nil {
		log.Fatal(err)
	}

	dflag := getDiscovery(c)
	if dflag == "" {
		log.Fatalf("discovery required to manage a cluster. See '%s manage --help'.", c.App.Name)
	}

	s, err := strategy.New(c.String("strategy"))
	if err != nil {
		log.Fatal(err)
	}

	// see https://github.com/codegangsta/cli/issues/160
	names := c.StringSlice("filter")
	if c.IsSet("filter") || c.IsSet("f") {
		names = names[DEFAULT_FILTER_NUMBER:]
	}
	fs, err := filter.New(names)
	if err != nil {
		log.Fatal(err)
	}

	sched := scheduler.New(s, fs)

	eventsHandler := api.NewEventsHandler()
	options := &cluster.Options{
		TLSConfig:       tlsConfig,
		OvercommitRatio: c.Float64("overcommit"),
		Discovery:       dflag,
		Heartbeat:       c.Int("heartbeat"),
	}

	cluster := swarm.NewCluster(sched, store, eventsHandler, options)

	// clean up all plugins
	defer strategy.StopPlugins()

	// see https://github.com/codegangsta/cli/issues/160
	hosts := c.StringSlice("host")
	if c.IsSet("host") || c.IsSet("H") {
		hosts = hosts[1:]
	}
	log.Fatal(api.ListenAndServe(cluster, hosts, c.Bool("cors"), tlsConfig, eventsHandler))
}
