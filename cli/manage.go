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
	"github.com/docker/docker/pkg/discovery"
	kvdiscovery "github.com/docker/docker/pkg/discovery/kv"
	"github.com/docker/leadership"
	"github.com/docker/libkv/store"
	"github.com/docker/swarm/api"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos"
	"github.com/docker/swarm/cluster/swarm"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
)

const (
	leaderElectionPath = "docker/swarm/leader"
	initialRecoverTime = 500 * time.Millisecond
	maxRecoverTime     = 10 * time.Second
)

type logHandler struct {
}

func (h *logHandler) Handle(e *cluster.Event) error {
	id := e.ID
	// Trim IDs to 12 chars.
	if len(id) > 12 {
		id = id[:12]
	}
	log.WithFields(log.Fields{"node": e.Engine.Name, "id": id, "from": e.From, "status": e.Status}).Debug("Event received")
	return nil
}

type statusHandler struct {
	cluster     cluster.Cluster
	candidate   *leadership.Candidate
	follower    *leadership.Follower
	addr        string
	recoverTime time.Duration
}

func (h *statusHandler) Status() [][2]string {
	var status [][2]string

	if h.candidate != nil && !h.candidate.IsLeader() && h.follower != nil {
		if h.follower.Leader() == h.addr {
			status = [][2]string{
				{"Role", "primary"},
			}
		} else {
			status = [][2]string{
				{"Role", "replica"},
				{"Primary", h.follower.Leader()},
			}
		}
	} else {
		status = [][2]string{
			{"Role", "primary"},
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
func createDiscovery(uri string, c *cli.Context) discovery.Backend {
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
	if _, ok := options["kv.path"]; !ok {
		options["kv.path"] = "docker/swarm/nodes"
	}
	return options
}

func setupReplication(c *cli.Context, cluster cluster.Cluster, handler *statusHandler, client store.Store, keyPath string, server *api.Server, addr string, tlsConfig *tls.Config, leaderTTL time.Duration) {
	primary := api.NewPrimary(cluster, tlsConfig, handler, c.GlobalBool("debug"), c.Bool("cors"))
	replica := api.NewReplica(primary, tlsConfig)

	go func() {
		for {
			err := run(cluster, client, keyPath, handler, server, primary, replica, addr, leaderTTL)
			if err == nil {
				handler.recoverTime = initialRecoverTime
			} else {
				// If we lost connection to the KV store, sleep
				// before reconnecting with an exponential
				// backoff.
				time.Sleep(handler.recoverTime)
				handler.recoverTime = 2 * handler.recoverTime
				if handler.recoverTime > maxRecoverTime {
					handler.recoverTime = maxRecoverTime
				}
			}
		}
	}()

	server.SetHandler(primary)
}

func run(cl cluster.Cluster, client store.Store, keyPath string, handler *statusHandler, server *api.Server, primary *mux.Router, replica *api.Replica, addr string, leaderTTL time.Duration) error {
	candidate := leadership.NewCandidate(client, keyPath, addr, leaderTTL)
	follower := leadership.NewFollower(client, keyPath)

	handler.candidate = candidate
	handler.follower = follower

	electedCh, candidateErrCh := candidate.RunForElection()
	leaderCh, followerErrCh := follower.FollowElection()
	defer follower.Stop()

	var watchdog *cluster.Watchdog
	wasLeader := false
	for {
		select {
		case isElected := <-electedCh:
			if isElected {
				log.Info("Leader Election: Cluster leadership acquired")
				wasLeader = true
				watchdog = cluster.NewWatchdog(cl)
				server.SetHandler(primary)
			} else {
				cl.UnregisterEventHandler(watchdog)
				// TODO(nishanttotla): perhaps EventHandler for subscription events should
				// also be unregistered here
				server.SetHandler(replica)
				if wasLeader {
					log.Info("Leader Election: Cluster leadership lost")
					candidate.Stop()
					return nil
				}
			}

		case leader := <-leaderCh:
			if leader == "" {
				continue
			}
			if leader == addr {
				replica.SetPrimary("")
				server.SetHandler(primary)
			} else {
				log.Infof("New leader elected: %s", leader)
				replica.SetPrimary(leader)
				server.SetHandler(replica)
			}

		case err := <-candidateErrCh:
			log.Errorf("Error from leadership election candidate: %s", err)
			return err

		case err := <-followerErrCh:
			log.Errorf("Error from leadership election follower: %s", err)
			return err
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

	refreshMinInterval := c.Duration("engine-refresh-min-interval")
	refreshMaxInterval := c.Duration("engine-refresh-max-interval")
	if refreshMinInterval <= time.Duration(0)*time.Second {
		log.Fatal("min refresh interval should be a positive number")
	}
	if refreshMaxInterval < refreshMinInterval {
		log.Fatal("max refresh interval cannot be less than min refresh interval")
	}
	// engine-refresh-retry is deprecated
	refreshRetry := c.Int("engine-refresh-retry")
	if refreshRetry != 3 {
		log.Fatal("--engine-refresh-retry is deprecated. Use --engine-failure-retry")
	}
	failureRetry := c.Int("engine-failure-retry")
	if failureRetry <= 0 {
		log.Fatal("invalid failure retry count")
	}
	engineOpts := &cluster.EngineOpts{
		RefreshMinInterval: refreshMinInterval,
		RefreshMaxInterval: refreshMaxInterval,
		FailureRetry:       failureRetry,
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

	// see https://github.com/urfave/cli/issues/160
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
		cl, err = mesos.NewCluster(sched, tlsConfig, uri, c.StringSlice("cluster-opt"), engineOpts)
	case "swarm":
		cl, err = swarm.NewCluster(sched, tlsConfig, discovery, c.StringSlice("cluster-opt"), engineOpts)
	default:
		log.Fatalf("unsupported cluster %q", c.String("cluster-driver"))
	}
	if err != nil {
		log.Fatal(err)
	}

	// see https://github.com/urfave/cli/issues/160
	hosts := c.StringSlice("host")
	if c.IsSet("host") || c.IsSet("H") {
		hosts = hosts[1:]
	}

	api.ShouldRefreshOnNodeFilter = c.Bool("refresh-on-node-filter")
	api.ContainerNameRefreshFilter = c.String("container-name-refresh-filter")

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
		if leaderTTL <= time.Duration(0)*time.Second {
			log.Fatalf("--replication-ttl should be a positive number")
		}

		kvDiscovery, ok := discovery.(*kvdiscovery.Discovery)
		if !ok {
			log.Fatal("Leader election is only supported with consul, etcd and zookeeper discovery.")
		}
		client := kvDiscovery.Store()
		keyPath := path.Join(kvDiscovery.Prefix(), leaderElectionPath)

		handler := &statusHandler{
			cluster:     cl,
			addr:        addr,
			recoverTime: initialRecoverTime,
		}
		defer func(h *statusHandler) {
			// Make sure we resign the leadership position when we
			// exit if necessary.
			if h.candidate != nil {
				h.candidate.Resign()
			}
		}(handler)

		setupReplication(c, cl, handler, client, keyPath, server, addr, tlsConfig, leaderTTL)
	} else {
		handler := &statusHandler{
			cluster:     cl,
			addr:        "",
			recoverTime: initialRecoverTime,
		}
		server.SetHandler(api.NewPrimary(cl, tlsConfig, handler, c.GlobalBool("debug"), c.Bool("cors")))
		cluster.NewWatchdog(cl)
	}
	defer cl.CloseWatchQueues()

	log.Fatal(server.ListenAndServe())
}
