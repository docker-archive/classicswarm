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

func (s *FileDiscoveryService) Fetch() ([]*discovery.Entry, error) {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var entries []*discovery.Entry

	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			entry, err := discovery.NewEntry(line)
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
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
