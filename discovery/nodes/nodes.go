package nodes

import (
	"strings"

	"github.com/docker/swarm/discovery"
)

type NodesDiscoveryService struct {
	entries []*discovery.Entry
}

func init() {
	discovery.Register("nodes", &NodesDiscoveryService{})
}

func (s *NodesDiscoveryService) Initialize(uris string, _ int) error {
	for _, input := range strings.Split(uris, ",") {
		for _, ip := range discovery.Generate(input) {
			entry, err := discovery.NewEntry(ip)
			if err != nil {
				return err
			}
			s.entries = append(s.entries, entry)
		}
	}

	return nil
}
func (s *NodesDiscoveryService) Fetch() ([]*discovery.Entry, error) {
	return s.entries, nil
}

func (s *NodesDiscoveryService) Watch(callback discovery.WatchCallback) {
}

func (s *NodesDiscoveryService) Register(addr string) error {
	return discovery.ErrNotImplemented
}
