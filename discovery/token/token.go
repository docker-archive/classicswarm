package token

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
)

const DISCOVERY_URL = "https://discovery-stage.hub.docker.com/v1"

type TokenDiscoveryService struct {
	heartbeat int
	url       string
	token     string
}

func init() {
	discovery.Register("token",
		func() discovery.DiscoveryService {
			return &TokenDiscoveryService{}
		},
	)
}

func (s *TokenDiscoveryService) Initialize(urltoken string, heartbeat int) error {
	if i := strings.LastIndex(urltoken, "/"); i != -1 {
		s.url = "https://" + urltoken[:i]
		s.token = urltoken[i+1:]
	} else {
		s.url = DISCOVERY_URL
		s.token = urltoken
	}
	s.heartbeat = heartbeat

	return nil
}

// FetchNodes returns the node for the discovery service at the specified endpoint
func (s *TokenDiscoveryService) Fetch() ([]*discovery.Node, error) {
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
	}

	var nodes []*discovery.Node
	for _, addr := range addrs {
		nodes = append(nodes, discovery.NewNode(addr))
	}

	return nodes, nil
}

func (s *TokenDiscoveryService) Watch(c *cluster.Cluster, refresh func(c *cluster.Cluster, nodes []*discovery.Node)) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		nodes, err := s.Fetch()
		if err == nil {
			refresh(c, nodes)
		}
	}
}

// RegisterNode adds a new node identified by the into the discovery service
func (s *TokenDiscoveryService) Register(addr string) error {
	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/%s/%s", s.url,
		"clusters", s.token), "application/json", buf)
	return err
}

// CreateCluster returns a unique cluster token
func (s *TokenDiscoveryService) CreateCluster() (string, error) {
	resp, err := http.Post(fmt.Sprintf("%s/%s", s.url, "clusters"), "", nil)
	if err != nil {
		return "", err
	}
	token, err := ioutil.ReadAll(resp.Body)
	return string(token), err
}
