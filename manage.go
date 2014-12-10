package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

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

	refresh := func(c *cluster.Cluster, nodes []string) {
		for _, addr := range nodes {
			go func(addr string) {
				if !strings.Contains(addr, "://") {
					addr = "http://" + addr
				}
				if c.Node(addr) == nil {
					n := cluster.NewNode(addr)
					if err := n.Connect(tlsConfig); err != nil {
						log.Error(err)
						return
					}
					if err := c.AddNode(n); err != nil {
						log.Error(err)
						return
					}
				}
			}(addr)
		}
	}

	cluster := cluster.NewCluster()
	cluster.Events(&logHandler{})

	go func() {
		fmt.Println(c.String("discovery"))
		if c.String("discovery") != "" {
			d, err := discovery.New(c.String("discovery"))
			if err != nil {
				log.Fatal(err)
			}

			nodes, err := d.FetchNodes()
			if err != nil {
				log.Fatal(err)

			}
			refresh(cluster, nodes)

			hb := time.Duration(c.Int("heartbeat"))
			go func() {
				for {
					time.Sleep(hb * time.Second)
					nodes, err = d.FetchNodes()
					if err == nil {
						refresh(cluster, nodes)
					}
				}
			}()
		} else {
			refresh(cluster, c.Args())
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
