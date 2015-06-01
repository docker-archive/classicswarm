package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos"
	"github.com/docker/swarm/cluster/swarm"
	"github.com/docker/swarm/discovery"
	kvdiscovery "github.com/docker/swarm/discovery/kv"
	"github.com/docker/swarm/leadership"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/docker/swarm/state"
)

const (
	leaderElectionPath = "docker/swarm/leader"
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	id := e.Id
	// Trim IDs to 12 chars.
	if len(id) > 12 {
		id = id[:12]
	}
	log.WithFields(log.Fields{"node": e.Engine.Name, "id": id, "from": e.From, "status": e.Status}).Debug("Event received")
	return nil
}

type statusHandler struct {
	cluster   cluster.Cluster
	candidate *leadership.Candidate
	follower  *leadership.Follower
}

func (h *statusHandler) Status() [][]string {
	var status [][]string

	if h.candidate != nil && !h.candidate.IsLeader() {
		status = [][]string{
			{"\bRole", "replica"},
			{"\bPrimary", h.follower.Leader()},
		}
	} else {
		status = [][]string{
			{"\bRole", "primary"},
		}
	}

	status = append(status, h.cluster.Info()...)
	return status
}

// Load the TLS certificates/keys and, if verify is true, the CA.
func loadTLSConfig(ca, cert, key string, verify bool) (*tls.Config, error) {
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

// Initialize the discovery service.
func createDiscovery(uri string, c *cli.Context) discovery.Discovery {
	hb, err := time.ParseDuration(c.String("heartbeat"))
	if err != nil {
		log.Fatalf("invalid --heartbeat: %v", err)
	}
	if hb < 1*time.Second {
		log.Fatal("--heartbeat should be at least one second")
	}

	// Set up discovery.
	discovery, err := discovery.New(uri, hb, 0)
	if err != nil {
		log.Fatal(err)
	}

	return discovery
}

func setupReplication(c *cli.Context, cluster cluster.Cluster, server *api.Server, discovery discovery.Discovery, addr string, tlsConfig *tls.Config) {
	kvDiscovery, ok := discovery.(*kvdiscovery.Discovery)
	if !ok {
		log.Fatal("Leader election is only supported with consul, etcd and zookeeper discovery.")
	}
	client := kvDiscovery.Store()

	candidate := leadership.NewCandidate(client, leaderElectionPath, addr)
	follower := leadership.NewFollower(client, leaderElectionPath)

	router := api.NewRouter(cluster, tlsConfig, &statusHandler{cluster, candidate, follower}, c.Bool("cors"))
	proxy := api.NewReverseProxy(router, tlsConfig)

	go func() {
		candidate.RunForElection()
		electedCh := candidate.ElectedCh()
		for isElected := range electedCh {
			if isElected {
				log.Info("Cluster leadership acquired")
				server.SetHandler(router)
			} else {
				log.Info("Cluster leadership lost")
				server.SetHandler(proxy)
			}
		}
	}()

	go func() {
		follower.FollowElection()
		leaderCh := follower.LeaderCh()
		for leader := range leaderCh {
			log.Infof("New leader elected: %s", leader)
			if leader == addr {
				proxy.SetDestination("")
			} else {
				proxy.SetDestination(leader)
			}
		}
	}()

	server.SetHandler(router)
}

func manage(c *cli.Context) {
	var (
		tlsConfig *tls.Config
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
		tlsConfig, err = loadTLSConfig(
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

	uri := getDiscovery(c)
	if uri == "" {
		log.Fatalf("discovery required to manage a cluster. See '%s manage --help'.", c.App.Name)
	}
	discovery := createDiscovery(uri, c)
	s, err := strategy.New(c.String("strategy"))
	if err != nil {
		log.Fatal(err)
	}

	// see https://github.com/codegangsta/cli/issues/160
	names := c.StringSlice("filter")
	if c.IsSet("filter") || c.IsSet("f") {
		names = names[DefaultFilterNumber:]
	}
	fs, err := filter.New(names)
	if err != nil {
		log.Fatal(err)
	}

	sched := scheduler.New(s, fs)
	var cl cluster.Cluster
	switch c.String("cluster-driver") {
	case "mesos-experimental":
		log.Warn("WARNING: the mesos driver is currently experimental, use at your own risks")
		cl, err = mesos.NewCluster(sched, store, tlsConfig, uri, c.StringSlice("cluster-opt"))
	case "swarm":
		cl, err = swarm.NewCluster(sched, store, tlsConfig, discovery, c.StringSlice("cluster-opt"))
	default:
		log.Fatalf("unsupported cluster %q", c.String("cluster-driver"))
	}
	if err != nil {
		log.Fatal(err)
	}

	// see https://github.com/codegangsta/cli/issues/160
	hosts := c.StringSlice("host")
	if c.IsSet("host") || c.IsSet("H") {
		hosts = hosts[1:]
	}

	server := api.NewServer(hosts, tlsConfig)
	if c.Bool("replication") {
		addr := c.String("advertise")
		if addr == "" {
			log.Fatal("--advertise address must be provided when using --leader-election")
		}
		if !checkAddrFormat(addr) {
			log.Fatal("--advertise should be of the form ip:port or hostname:port")
		}

		setupReplication(c, cl, server, discovery, addr, tlsConfig)
	} else {
		server.SetHandler(api.NewRouter(cl, tlsConfig, &statusHandler{cl, nil, nil}, c.Bool("cors")))
	}

	log.Fatal(server.ListenAndServe())
}
