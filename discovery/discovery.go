package discovery

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
)

type InitFunc func(url string) (DiscoveryService, error)

type DiscoveryService interface {
	Fetch() ([]string, error)
	Watch(int) <-chan time.Time
	Register(string) error
}

var (
	discoveries     map[string]InitFunc
	ErrNotSupported = errors.New("discovery service not supported")
)

func init() {
	discoveries = make(map[string]InitFunc)
}

func Register(scheme string, initFunc InitFunc) error {
	if _, exists := discoveries[scheme]; exists {
		return fmt.Errorf("scheme already registered %s", scheme)
	}
	log.Debugf("Registering %q discovery service", scheme)
	discoveries[scheme] = initFunc

	return nil
}

func New(rawurl string) (DiscoveryService, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if initFct, exists := discoveries[url.Scheme]; exists {
		log.Debugf("Initialising %q discovery service with %q", url.Scheme, url.Host+url.Path)
		return initFct(url.Host + url.Path)
	}

	return nil, ErrNotSupported
}
