package discovery

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Node struct {
	url string
}

func NewNode(url string) *Node {
	return &Node{url: url}
}

func (n Node) String() string {
	return n.url
}

type WatchCallback func(nodes []*Node)

type DiscoveryService interface {
	Initialize(string, int) error
	Fetch() ([]*Node, error)
	Watch(WatchCallback)
	Register(string) error
}

var (
	discoveries       map[string]DiscoveryService
	ErrNotSupported   = errors.New("discovery service not supported")
	ErrNotImplemented = errors.New("not implemented in this discovery service")
)

func init() {
	discoveries = make(map[string]DiscoveryService)
}

func Register(scheme string, d DiscoveryService) error {
	if _, exists := discoveries[scheme]; exists {
		return fmt.Errorf("scheme already registered %s", scheme)
	}
	log.Debugf("Registering %q discovery service", scheme)
	discoveries[scheme] = d

	return nil
}

func parse(rawurl string) (string, string) {
	parts := strings.SplitN(rawurl, "://", 2)

	// nodes:port,node2:port => nodes://node1:port,node2:port
	if len(parts) == 1 {
		return "nodes", parts[0]
	}
	return parts[0], parts[1]
}

func New(rawurl string, heartbeat int) (DiscoveryService, error) {
	scheme, uri := parse(rawurl)

	if discovery, exists := discoveries[scheme]; exists {
		log.Debugf("Initializing %q discovery service with %q", scheme, uri)
		err := discovery.Initialize(uri, heartbeat)
		return discovery, err
	}

	return nil, ErrNotSupported
}
