package discovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	DOCKER_HUB_SCHEME     = "hub://"
	DISCOVERY_URL_DEFAULT = "https://discovery-stage.hub.docker.com/v1"
)

type DockerHubDiscovery struct {
}

func setupUrl(url string) string {
	if url == DOCKER_HUB_SCHEME {
		return DISCOVERY_URL_DEFAULT
	}

	if strings.HasPrefix(url, DOCKER_HUB_SCHEME) {
		newUrl := url[len(DOCKER_HUB_SCHEME):]

		if strings.HasPrefix(newUrl, "http://") || strings.HasPrefix(newUrl, "https://") {
			return newUrl
		}

		return "https://" + newUrl
	}

	return ""
}

func (d *DockerHubDiscovery) Supports(url string) (bool, error) {
	return len(setupUrl(url)) > 0, nil
}

// FetchSlaves returns the slaves for the discovery service at the specified endpoint
func (d *DockerHubDiscovery) FetchSlaves(url, token string) ([]string, error) {
	url = setupUrl(url)

	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", url, "clusters", token))
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
func (d *DockerHubDiscovery) RegisterSlave(url, addr, token string) error {
	url = setupUrl(url)

	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/%s/%s", url, "clusters", token), "application/json", buf)
	return err
}

// CreateCluster returns a unique cluster token
func (d *DockerHubDiscovery) CreateCluster(url string) (string, error) {
	url = setupUrl(url)

	resp, err := http.Post(fmt.Sprintf("%s/%s", url, "clusters"), "", nil)
	if err != nil {
		return "", err
	}
	token, err := ioutil.ReadAll(resp.Body)
	return string(token), err
}
