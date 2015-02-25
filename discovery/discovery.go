package discovery

import (
	"errors"
	"fmt"
	"net"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Entry struct {
	Host string
	Port string
}

func NewEntry(url string) (*Entry, error) {
	host, port, err := net.SplitHostPort(url)
	if err != nil {
		return nil, err
	}
	return &Entry{host, port}, nil
}

func (m Entry) String() string {
	return fmt.Sprintf("%s:%s", m.Host, m.Port)
}

type WatchCallback func(entries []*Entry)

type DiscoveryService interface {
	Initialize(string, int) error
	Fetch() ([]*Entry, error)
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
	log.WithField("name", scheme).Debug("Registering discovery service")
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
		log.WithFields(log.Fields{"name": scheme, "uri": uri}).Debug("Initializing discovery service")
		err := discovery.Initialize(uri, heartbeat)
		return discovery, err
	}

	return nil, ErrNotSupported
}

func CreateEntries(addrs []string) ([]*Entry, error) {
	entries := []*Entry{}
	if addrs == nil {
		return entries, nil
	}

	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		entry, err := NewEntry(addr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
