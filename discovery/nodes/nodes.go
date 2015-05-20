package nodes

import (
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

// Discovery is exported
type Discovery struct {
	entries discovery.Entries
}

func init() {
	discovery.Register("nodes", &Discovery{})
}

// Initialize is exported
func (s *Discovery) Initialize(uris string, _ time.Duration, _ time.Duration) error {
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

// Watch is exported
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	go func() {
		defer close(ch)
		ch <- s.entries
		<-stopCh
	}()
	return ch, nil
}

// Register is exported
func (s *Discovery) Register(addr string) error {
	return discovery.ErrNotImplemented
}
