package discovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const DISCOVERY_URL = "https://discovery-stage.hub.docker.com/v1"

// FetchSlaves returns the slaves for the discovery service at the specified endpoint
func FetchSlaves(token string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", DISCOVERY_URL, "clusters", token))
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

// RegisterSlave adds a new slave identified by the slaveID into the discovery service
// the default TTL is 30 secs
func RegisterSlave(addr, token string) error {
	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/%s/%s", DISCOVERY_URL, "clusters", token), "application/json", buf)
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
