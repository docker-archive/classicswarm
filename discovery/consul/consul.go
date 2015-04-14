package consul

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/discovery"
	consul "github.com/hashicorp/consul/api"
)

var (
	// ErrConsulNotReachable thrown when no consul agent's api is reachable
	ErrConsulNotReachable = errors.New("No consul agents reachable")
	defaultConsulPort     = "8500"
	defaultWaitTime       = 5
)

// ConsulDiscoveryService is exported
type ConsulDiscoveryService struct {
	heartbeat time.Duration
	client    *consul.Client
	agents    string
	prefix    string
	lastIndex uint64
}

func init() {
	discovery.Register("consul", &ConsulDiscoveryService{})
}

// Initialize is exported
func (s *ConsulDiscoveryService) Initialize(uris string, heartbeat uint64) error {
	parts := strings.SplitN(uris, "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid format %q, missing <path>", uris)
	}

	// Split multiple IPs
	ips := strings.Split(parts[0], ",")
	path := parts[1]
	s.agents = uris

	for _, ip := range ips {
		err := s.initConsulClient(ip, heartbeat, path)
		// If unreachable we continue to the next ip
		if err != nil {
			log.WithField("name", "consul").Debug("Cannot connect to consul node: ", ip)
			continue
		}

		// Create new K/V client
		kv := s.client.KV()
		p := &consul.KVPair{Key: s.prefix, Value: nil}

		_, meta, err := kv.Get(s.prefix, nil)
		if err == nil {
			// Value already exists due to switching agent context
			// update index and skip
			log.WithField("name", "consul").Info("Connection established with consul agent: ", ip)
			s.lastIndex = meta.LastIndex
			return nil
		}
		if _, err = kv.Put(p, nil); err != nil {
			log.WithField("name", "consul").Error(err)
			continue
		}
		_, meta, err = kv.Get(s.prefix, nil)
		if err != nil {
			log.WithField("name", "consul").Error(err)
			continue
		}
		s.lastIndex = meta.LastIndex
		return nil
	}

	return ErrConsulNotReachable
}

func (s *ConsulDiscoveryService) initConsulClient(ip string, heartbeat int, prefix string) error {
	addr := fmt.Sprint(ip, ":", defaultConsulPort)
	config := &consul.Config{
		Address:    addr,
		Scheme:     "http",
		HttpClient: http.DefaultClient,
		WaitTime:   time.Duration(defaultWaitTime) * time.Second,
	}
	client, err := consul.NewClient(config)
	if err != nil {
		return err
	}
	s.client = client
	s.heartbeat = time.Duration(heartbeat) * time.Second
	s.prefix = prefix + "/"
	return nil
}

func (s *ConsulDiscoveryService) switchConsulAgent() error {
	log.WithField("name", "consul").Debug("Consul unreachable, retrying and switching agent..")
	return s.Initialize(s.agents, int(s.heartbeat.Seconds()))
}

// Fetch is exported
func (s *ConsulDiscoveryService) Fetch() ([]*discovery.Entry, error) {
RETRY:
	kv := s.client.KV()
	pairs, _, err := kv.List(s.prefix, nil)
	if err != nil {
		err = s.switchConsulAgent()
		if err != nil {
			return nil, err
		}
		goto RETRY
	}

	addrs := []string{}
	for _, pair := range pairs {
		if pair.Key == s.prefix {
			continue
		}
		addrs = append(addrs, string(pair.Value))
	}

	return discovery.CreateEntries(addrs)
}

// Watch is exported
func (s *ConsulDiscoveryService) Watch(callback discovery.WatchCallback) {
	for _ = range s.waitForChange() {
		log.WithField("name", "consul").Debug("Discovery watch triggered")
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

// Register is exported
func (s *ConsulDiscoveryService) Register(addr string) error {
	kv := s.client.KV()
	p := &consul.KVPair{Key: path.Join(s.prefix, addr), Value: []byte(addr)}
	_, err := kv.Put(p, nil)
	if err != nil {
		err = s.switchConsulAgent()
		if err != nil {
			return ErrConsulNotReachable
		}
	}
	return nil
}

func (s *ConsulDiscoveryService) waitForChange() <-chan uint64 {
	c := make(chan uint64)
	go func() {
		for {
			kv := s.client.KV()
			option := &consul.QueryOptions{
				WaitIndex: s.lastIndex,
				WaitTime:  s.heartbeat}
			_, meta, err := kv.List(s.prefix, option)
			if err != nil {
				err = s.switchConsulAgent()
				if err != nil {
					log.WithField("name", "consul").Errorf("Discovery error: %v", err)
					break
				}
				continue
			}
			s.lastIndex = meta.LastIndex
			c <- s.lastIndex
		}
		close(c)
	}()
	return c
}
