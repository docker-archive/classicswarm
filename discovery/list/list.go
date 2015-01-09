package list

import (
	"strings"

	"github.com/docker/swarm/discovery"
)

type ListDiscoveryService struct {
	list []*discovery.Node
}

func init() {
	discovery.Register("list", &ListDiscoveryService{})
}

func (s *ListDiscoveryService) Initialize(uris string, _ int) error {
	for _, ip := range strings.Split(uris, ",") {
		s.list = append(s.list, discovery.NewNode(ip))
	}

	return nil
}
func (s *ListDiscoveryService) Fetch() ([]*discovery.Node, error) {
	return s.list, nil
}

func (s *ListDiscoveryService) Watch(callback discovery.WatchCallback) {
}

func (s *ListDiscoveryService) Register(addr string) error {
	return discovery.ErrNotImplemented
}
