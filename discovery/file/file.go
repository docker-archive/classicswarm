package file

import (
	"errors"
	"io/ioutil"
	"strings"
	"time"

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

func (s FileDiscoveryService) Fetch() ([]string, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), "\n"), nil
}

func (s FileDiscoveryService) Watch(heartbeat int) <-chan time.Time {
	return time.Tick(time.Duration(heartbeat) * time.Second)
}

func (s FileDiscoveryService) Register(addr string) error {
	return errors.New("unimplemented")
}
