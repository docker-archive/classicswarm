package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

const DISCOVERY_URL = "https://discovery-stage.hub.docker.com/v1"

type TokenDiscoveryService struct {
	heartbeat int
	url       string
	token     string
}

func init() {
	discovery.Register("token", &TokenDiscoveryService{})
}

func (s *TokenDiscoveryService) Initialize(urltoken string, heartbeat int) error {
	if i := strings.LastIndex(urltoken, "/"); i != -1 {
		s.url = "https://" + urltoken[:i]
		s.token = urltoken[i+1:]
	} else {
		s.url = DISCOVERY_URL
		s.token = urltoken
	}

	if s.token == "" {
		return errors.New("token is empty")
	}
	s.heartbeat = heartbeat

	return nil
}

// Fetch returns the list of entries for the discovery service at the specified endpoint
func (s *TokenDiscoveryService) Fetch() ([]*discovery.Entry, error) {

	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", s.url, "clusters", s.token))
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var addrs []string
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&addrs); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Failed to fetch entries, Discovery service returned %d HTTP status code", resp.StatusCode)
	}

	return discovery.CreateEntries(addrs)
}

func (s *TokenDiscoveryService) Watch(callback discovery.WatchCallback) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

// RegisterEntry adds a new entry identified by the into the discovery service
func (s *TokenDiscoveryService) Register(addr string) error {
	buf := strings.NewReader(addr)

	resp, err := http.Post(fmt.Sprintf("%s/%s/%s", s.url,
		"clusters", s.token), "application/json", buf)
	if err == nil {
		// Force connection close
		resp.Body.Close()
	}
	return err
}

// CreateCluster returns a unique cluster token
func (s *TokenDiscoveryService) CreateCluster() (string, error) {
	resp, err := http.Post(fmt.Sprintf("%s/%s", s.url, "clusters"), "", nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	token, err := ioutil.ReadAll(resp.Body)
	return string(token), err
}
