package file

import (
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
	discovery.Register("file", &FileDiscoveryService{})
}

func (s *FileDiscoveryService) Initialize(path string, heartbeat int) error {
	s.path = path
	s.heartbeat = heartbeat
	return nil
}

func parseFileContent(content []byte) []string {
	result := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		for _, ip := range discovery.Generate(line) {
			result = append(result, ip)
		}
	}
	return result
}

func (s *FileDiscoveryService) Fetch() ([]*discovery.Entry, error) {
	fileContent, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	return discovery.CreateEntries(parseFileContent(fileContent))
}

func (s *FileDiscoveryService) Watch(callback discovery.WatchCallback) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

func (s *FileDiscoveryService) Register(addr string) error {
	return discovery.ErrNotImplemented
}
