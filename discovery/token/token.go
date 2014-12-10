package token

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

const DISCOVERY_URL = "https://discovery-stage.hub.docker.com/v1"

type TokenDiscoveryService struct {
	token string
}

func init() {
	discovery.Register("token", Init)
}

func Init(token string) (discovery.DiscoveryService, error) {
	return TokenDiscoveryService{token: token}, nil
}

// FetchNodes returns the node for the discovery service at the specified endpoint
func (s TokenDiscoveryService) Fetch() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", DISCOVERY_URL, "clusters", s.token))
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

	return addrs, nil
}

func (s TokenDiscoveryService) Watch(heartbeat int) <-chan time.Time {
	return time.Tick(time.Duration(heartbeat) * time.Second)
}

// RegisterNode adds a new node identified by the into the discovery service
func (s TokenDiscoveryService) Register(addr string) error {
	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/%s/%s", DISCOVERY_URL,
		"clusters", s.token), "application/json", buf)
	return err
}

// CreateCluster returns a unique cluster token
func CreateCluster() (string, error) {
	resp, err := http.Post(fmt.Sprintf("%s/%s", DISCOVERY_URL, "clusters"), "", nil)
	if err != nil {
		return "", err
	}
	token, err := ioutil.ReadAll(resp.Body)
	return string(token), err
}
