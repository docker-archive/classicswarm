package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
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
	"github.com/gorilla/mux"
)

const (
	leaderElectionPath = "docker/swarm/leader"
	defaultRecoverTime = 10 * time.Second
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
func createDiscovery(uri string, c *cli.Context, discoveryOpt []string) discovery.Discovery {
	hb, err := time.ParseDuration(c.String("heartbeat"))
	if err != nil {
		log.Fatalf("invalid --heartbeat: %v", err)
	}
	if hb < 1*time.Second {
		log.Fatal("--heartbeat should be at least one second")
	}

	// Set up discovery.
	discovery, err := discovery.New(uri, hb, 0, getDiscoveryOpt(c))
	if err != nil {
		log.Fatal(err)
	}

	return discovery
}

func getDiscoveryOpt(c *cli.Context) map[string]string {
	// Process the store options
	options := map[string]string{}
	for _, option := range c.StringSlice("discovery-opt") {
		if !strings.Contains(option, "=") {
			log.Fatal("--discovery-opt must contain key=value strings")
		}
		kvpair := strings.SplitN(option, "=", 2)
		options[kvpair[0]] = kvpair[1]
	}
	return options
}

func setupReplication(c *cli.Context, cluster cluster.Cluster, server *api.Server, discovery discovery.Discovery, addr string, leaderTTL time.Duration, tlsConfig *tls.Config) {
	kvDiscovery, ok := discovery.(*kvdiscovery.Discovery)
	if !ok {
		log.Fatal("Leader election is only supported with consul, etcd and zookeeper discovery.")
	}
	client := kvDiscovery.Store()
	p := path.Join(kvDiscovery.Prefix(), leaderElectionPath)

	candidate := leadership.NewCandidate(client, p, addr, leaderTTL)
	follower := leadership.NewFollower(client, p)

	primary := api.NewPrimary(cluster, tlsConfig, &statusHandler{cluster, candidate, follower}, c.Bool("cors"))
	replica := api.NewReplica(primary, tlsConfig)

	go func() {
		for {
			run(candidate, server, primary, replica)
			time.Sleep(defaultRecoverTime)
		}
	}()

	go func() {
		for {
			follow(follower, replica, addr)
			time.Sleep(defaultRecoverTime)
		}
	}()

	server.SetHandler(primary)
}

func run(candidate *leadership.Candidate, server *api.Server, primary *mux.Router, replica *api.Replica) {
	electedCh, errCh := candidate.RunForElection()
	for {
		select {
		case isElected := <-electedCh:
			if isElected {
				log.Info("Leader Election: Cluster leadership acquired")
				server.SetHandler(primary)
			} else {
				log.Info("Leader Election: Cluster leadership lost")
				server.SetHandler(replica)
			}

		case err := <-errCh:
			log.Error(err)
			return
		}
	}
}

func follow(follower *leadership.Follower, replica *api.Replica, addr string) {
	leaderCh, errCh := follower.FollowElection()
	for {
		select {
		case leader := <-leaderCh:
			if leader == "" {
				continue
			}
			if leader == addr {
				replica.SetPrimary("")
			} else {
				log.Infof("New leader elected: %s", leader)
				replica.SetPrimary(leader)
			}

		case err := <-errCh:
			log.Error(err)
			return
		}
	}
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

	uri := getDiscovery(c)
	if uri == "" {
		log.Fatalf("discovery required to manage a cluster. See '%s manage --help'.", c.App.Name)
	}
	discovery := createDiscovery(uri, c, c.StringSlice("discovery-opt"))
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
		cl, err = mesos.NewCluster(sched, tlsConfig, uri, c.StringSlice("cluster-opt"))
	case "swarm":
		cl, err = swarm.NewCluster(sched, tlsConfig, discovery, c.StringSlice("cluster-opt"))
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
		leaderTTL, err := time.ParseDuration(c.String("replication-ttl"))
		if err != nil {
			log.Fatalf("invalid --replication-ttl: %v", err)
		}

		setupReplication(c, cl, server, discovery, addr, leaderTTL, tlsConfig)
	} else {
		server.SetHandler(api.NewPrimary(cl, tlsConfig, &statusHandler{cl, nil, nil}, c.Bool("cors")))
	}

	log.Fatal(server.ListenAndServe())
}
