package kv

import (
	"fmt"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/pkg/store"
)

// Discovery is exported
type Discovery struct {
	backend   store.Backend
	store     store.Store
	heartbeat time.Duration
	prefix    string
}

func init() {
	discovery.Register("zk", &Discovery{backend: store.ZK})
	discovery.Register("consul", &Discovery{backend: store.CONSUL})
	discovery.Register("etcd", &Discovery{backend: store.ETCD})
}

// Initialize is exported
func (s *Discovery) Initialize(uris string, heartbeat uint64) error {
	var (
		parts = strings.SplitN(uris, "/", 2)
		ips   = strings.Split(parts[0], ",")
		addrs []string
		err   error
	)

	if len(parts) != 2 {
		return fmt.Errorf("invalid format %q, missing <path>", uris)
	}

	for _, ip := range ips {
		addrs = append(addrs, ip)
	}

	s.heartbeat = time.Duration(heartbeat) * time.Second
	s.prefix = parts[1]

	// Creates a new store, will ignore options given
	// if not supported by the chosen store
	s.store, err = store.CreateStore(
		s.backend,
		addrs,
		&store.Config{
			Timeout: s.heartbeat,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// Watch the store until either there's a store error or we receive a stop request.
// Returns false if we shouldn't attempt watching the store anymore (stop request received).
func (s *Discovery) watchOnce(stopCh <-chan struct{}, watchCh <-chan []*store.KVPair, discoveryCh chan discovery.Entries) bool {
	for {
		select {
		case pairs := <-watchCh:
			if pairs == nil {
				return true
			}

			log.WithField("discovery", s.backend).Debugf("Watch triggered with %d nodes", len(pairs))

			// Convert `KVPair` into `discovery.Entry`.
			addrs := make([]string, len(pairs))
			for _, pair := range pairs {
				addrs = append(addrs, string(pair.Value))
			}

			if entries, err := discovery.CreateEntries(addrs); err == nil {
				discoveryCh <- entries
			}
		case <-stopCh:
			// We were requested to stop watching.
			return false
		}
	}
}

// Watch is exported
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, error) {
	ch := make(chan discovery.Entries)
	go func() {
		// Forever: Create a store watch, watch until we get an error and then try again.
		// Will only stop if we receive a stopCh request.
		for {
			// Set up a watch.
			watchCh, err := s.store.WatchTree(s.prefix, stopCh)
			if err != nil {
				log.WithField("discovery", s.backend).Errorf("Unable to set up a watch: %v", err)
			} else {
				if !s.watchOnce(stopCh, watchCh, ch) {
					log.WithField("discovery", s.backend).Infof("Shutting down")
					return
				}
			}

			// If we get here it means the store watch channel was closed. This
			// is unexpected so let's retry later.
			log.WithField("discovery", s.backend).Errorf("Unexpected watch error. Retrying in %s", time.Duration(s.heartbeat))
			time.Sleep(s.heartbeat)
		}
	}()

	return ch, nil
}

// Register is exported
func (s *Discovery) Register(addr string) error {
	return s.store.Put(path.Join(s.prefix, addr), []byte(addr))
}
