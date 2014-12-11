package file

import (
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

type FileDiscoveryService struct {
	heartbeat int
	path      string
}

func init() {
	discovery.Register("file", Init)
}

func Init(file string, heartbeat int) (discovery.DiscoveryService, error) {
	return FileDiscoveryService{path: file, heartbeat: heartbeat}, nil
}

func (s FileDiscoveryService) Fetch() ([]*discovery.Node, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var nodes []*discovery.Node

	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			nodes = append(nodes, discovery.NewNode(line))
		}
	}
	return nodes, nil
}

func (s FileDiscoveryService) Watch() <-chan time.Time {
	return time.Tick(time.Duration(s.heartbeat) * time.Second)
}

func (s FileDiscoveryService) Register(addr string) error {
	return errors.New("unimplemented")
}
