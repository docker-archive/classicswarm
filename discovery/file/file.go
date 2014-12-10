package file

import (
	"io/ioutil"
	"strings"

	"github.com/docker/swarm/discovery"
)

type FileDiscoveryService struct {
	path string
}

func init() {
	discovery.Register("file", Init)
}

func Init(file string) (discovery.DiscoveryService, error) {
	return FileDiscoveryService{path: file}, nil
}

func (s FileDiscoveryService) FetchNodes() ([]string, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), "\n"), nil
}

func (s FileDiscoveryService) RegisterNode(addr string) error {

	return nil
}
