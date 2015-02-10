package etcd

import (
	"fmt"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/docker/swarm/discovery"
)

type EtcdDiscoveryService struct {
	ttl    uint64
	client *etcd.Client
	path   string
}

func init() {
	discovery.Register("etcd", &EtcdDiscoveryService{})
}

func (s *EtcdDiscoveryService) Initialize(uris string, heartbeat int) error {
	var (
		// split here because uris can contain multiples ips
		// like `etcd://192.168.0.1,192.168.0.2,192.168.0.3/path`
		parts   = strings.SplitN(uris, "/", 2)
		ips     = strings.Split(parts[0], ",")
		entries []string
	)

	if len(parts) != 2 {
		return fmt.Errorf("invalid format %q, missing <path>", uris)
	}

	for _, ip := range ips {
		entries = append(entries, "http://"+ip)
	}

	s.client = etcd.NewClient(entries)
	s.ttl = uint64(heartbeat * 3 / 2)
	s.path = "/" + parts[1] + "/"
	if _, err := s.client.CreateDir(s.path, s.ttl); err != nil {
		if etcdError, ok := err.(*etcd.EtcdError); ok {
			if etcdError.ErrorCode != 105 { // skip key already exists
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
func (s *EtcdDiscoveryService) Fetch() ([]*discovery.Entry, error) {
	resp, err := s.client.Get(s.path, true, true)
	if err != nil {
		return nil, err
	}

	addrs := []string{}
	for _, n := range resp.Node.Nodes {
		addrs = append(addrs, n.Value)
	}
	return discovery.CreateEntries(addrs)
}

func (s *EtcdDiscoveryService) Watch(callback discovery.WatchCallback) {
	watchChan := make(chan *etcd.Response)
	go s.client.Watch(s.path, 0, true, watchChan, nil)
	for _ = range watchChan {
		log.WithField("name", "etcd").Debug("Discovery watch triggered")
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

func (s *EtcdDiscoveryService) Register(addr string) error {
	_, err := s.client.Set(path.Join(s.path, addr), addr, s.ttl)
	return err
}
