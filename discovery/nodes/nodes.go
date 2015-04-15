package nodes

import (
	"strings"

	"github.com/docker/swarm/discovery"
)

// DiscoveryService is exported
type DiscoveryService struct {
	entries []*discovery.Entry
}

func init() {
	discovery.Register("nodes", &DiscoveryService{})
}

// Initialize is exported
func (s *DiscoveryService) Initialize(uris string, _ uint64) error {
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

// Fetch is exported
func (s *DiscoveryService) Fetch() ([]*discovery.Entry, error) {
	return s.entries, nil
}

// Watch is exported
func (s *DiscoveryService) Watch(callback discovery.WatchCallback) {
}

// Register is exported
func (s *DiscoveryService) Register(addr string) error {
	return discovery.ErrNotImplemented
}
