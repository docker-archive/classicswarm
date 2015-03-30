package file

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

// FileDiscoveryService is exported
type FileDiscoveryService struct {
	heartbeat int
	path      string
}

func init() {
	discovery.Register("file", &FileDiscoveryService{})
}

// Initialize is exported
func (s *FileDiscoveryService) Initialize(path string, heartbeat int) error {
	s.path = path
	s.heartbeat = heartbeat
	return nil
}

func parseFileContent(content []byte) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		for _, ip := range discovery.Generate(line) {
			result = append(result, ip)
		}
	}
	return result
}

// Fetch is exported
func (s *FileDiscoveryService) Fetch() ([]*discovery.Entry, error) {
	fileContent, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	return discovery.CreateEntries(parseFileContent(fileContent))
}

// Watch is exported
func (s *FileDiscoveryService) Watch(callback discovery.WatchCallback) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

// Register is exported
func (s *FileDiscoveryService) Register(addr string) error {
	return discovery.ErrNotImplemented
}
