package kv

import (
	"fmt"
	"path"
	"strings"
	"time"

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

// Fetch is exported
func (s *Discovery) Fetch() ([]*discovery.Entry, error) {
	addrs, err := s.store.List(s.prefix)
	if err != nil {
		return nil, err
	}
	return discovery.CreateEntries(convertToStringArray(addrs))
}

// Watch is exported
func (s *Discovery) Watch(callback discovery.WatchCallback) {
	s.store.WatchTree(s.prefix, "", s.heartbeat, func(kv ...*store.KVPair) {
		// Traduce byte array entries to discovery.Entry
		entries, _ := discovery.CreateEntries(convertToStringArray(kv))
		callback(entries)
	})
}

// Register is exported
func (s *Discovery) Register(addr string) error {
	err := s.store.Put(path.Join(s.prefix, addr), []byte(addr))
	return err
}

func convertToStringArray(entries []*store.KVPair) (addrs []string) {
	for _, entry := range entries {
		addrs = append(addrs, string(entry.Value))
	}
	return addrs
}
