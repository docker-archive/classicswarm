package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	log.Printf("event -> status: %q from: %q id: %q node: %q", e.Status, e.From, e.Id, e.NodeName)
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

	cluster := cluster.NewCluster(tlsConfig)
	cluster.Events(&logHandler{})

	go func() {
		if c.String("discovery") != "" {
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
		} else {
			var nodes []*discovery.Node
			for _, arg := range c.Args() {
				nodes = append(nodes, discovery.NewNode(arg))
			}
			cluster.UpdateNodes(nodes)
		}
	}()

	s := scheduler.NewScheduler(
		cluster,
		&strategy.BinPackingPlacementStrategy{OvercommitRatio: 0.05},
		[]filter.Filter{
			&filter.HealthFilter{},
			&filter.LabelFilter{},
			&filter.PortFilter{},
		},
	)

	log.Fatal(api.ListenAndServe(cluster, s, c.String("addr"), c.App.Version, c.Bool("cors"), tlsConfig))
}
