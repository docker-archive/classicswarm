package discovery

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Node struct {
	url string
}

func NewNode(url string) *Node {
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	return &Node{url: url}
}

func (n Node) String() string {
	return n.url
}

type DiscoveryService interface {
	Initialize(string, int) error
	Fetch() ([]*Node, error)
	Watch(func(nodes []*Node))
	Register(string) error
}

var (
	discoveries     map[string]func() DiscoveryService
	ErrNotSupported = errors.New("discovery service not supported")
)

func init() {
	discoveries = make(map[string]func() DiscoveryService)
}

func Register(scheme string, f func() DiscoveryService) error {
	if _, exists := discoveries[scheme]; exists {
		return fmt.Errorf("scheme already registered %s", scheme)
	}
	log.Debugf("Registering %q discovery service", scheme)
	discoveries[scheme] = f

	return nil
}

func New(rawurl string, heartbeat int) (DiscoveryService, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if f, exists := discoveries[url.Scheme]; exists {
		log.Debugf("Initialising %q discovery service with %q", url.Scheme, url.Host+url.Path)
		discovery := f()
		err := discovery.Initialize(url.Host+url.Path, heartbeat)
		return discovery, err
	}

	return nil, ErrNotSupported
}
