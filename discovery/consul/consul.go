package consul

import (
	"errors"
	"path"
	"strings"
	"time"

	consul "github.com/armon/consul-api"
	"github.com/docker/swarm/discovery"
)

type ConsulDiscoveryService struct {
	heartbeat uint64
	client    *consul.Client
	prefix    string
}

func init() {
	discovery.Register("consul", &ConsulDiscoveryService{})
}

func (s *ConsulDiscoveryService) Initialize(uris string, heartbeat int) error {
	parts := strings.SplitN(uris, "/", 2)
	if len(parts) < 2 {
		return errors.New("missing consul prefix")
	}
	addr := parts[0]
	path := parts[1]

	config := consul.DefaultConfig()
	config.Address = addr

	client, err := consul.NewClient(config)
	if err != nil {
		return err
	}
	s.client = client
	s.heartbeat = uint64(heartbeat)
	s.prefix = path + "/"
	kv := s.client.KV()
	p := &consul.KVPair{Key: s.prefix, Value: nil}
	_, err = kv.Put(p, nil)
	if err != nil {
		return err
	}
	return nil
}
func (s *ConsulDiscoveryService) Fetch() ([]*discovery.Node, error) {
	kv := s.client.KV()
	pairs, _, err := kv.List(s.prefix, nil)
	if err != nil {
		return nil, err
	}

	var nodes []*discovery.Node

	for _, pair := range pairs {
		if pair.Key == s.prefix {
			continue
		}
		nodes = append(nodes, discovery.NewNode(string(pair.Value)))
	}
	return nodes, nil
}

func (s *ConsulDiscoveryService) Watch(callback discovery.WatchCallback) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		nodes, err := s.Fetch()
		if err == nil {
			callback(nodes)
		}
	}
}

func (s *ConsulDiscoveryService) Register(addr string) error {
	kv := s.client.KV()
	p := &consul.KVPair{Key: path.Join(s.prefix, addr), Value: []byte(addr)}
	_, err := kv.Put(p, nil)
	if err != nil {
		return err
	}
	return nil
}
