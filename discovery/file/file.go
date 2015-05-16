package file

import (
	"io/ioutil"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/discovery"
)

// Discovery is exported
type Discovery struct {
	heartbeat uint64
	path      string
}

func init() {
	discovery.Register("file", &Discovery{})
}

// Initialize is exported
func (s *Discovery) Initialize(path string, heartbeat uint64) error {
	s.path = path
	s.heartbeat = heartbeat
	return nil
}

func parseFileContent(content []byte) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		line = strings.TrimSpace(line)
		// Ignoring line starts with #
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Inlined # comment also ignored.
		if strings.Contains(line, "#") {
			line = line[0:strings.Index(line, "#")]
			// Trim additional spaces caused by above stripping.
			line = strings.TrimSpace(line)
		}
		for _, ip := range discovery.Generate(line) {
			result = append(result, ip)
		}
	}
	return result
}

func (s *Discovery) fetch() (discovery.Entries, error) {
	fileContent, err := ioutil.ReadFile(s.path)
	if err != nil {
		log.WithField("discovery", "file").Errorf("Failed to read '%s': %v", s.path, err)
		return nil, err
	}
	return discovery.CreateEntries(parseFileContent(fileContent))
}

// Watch is exported
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, error) {
	ch := make(chan discovery.Entries)
	ticker := time.NewTicker(time.Duration(s.heartbeat) * time.Second)

	go func() {
		// Send the initial entries if available.
		currentEntries, err := s.fetch()
		if err == nil {
			ch <- currentEntries
		}

		// Periodically send updates.
		for {
			select {
			case <-ticker.C:
				newEntries, err := s.fetch()
				if err != nil {
					continue
				}

				// Check if the file has really changed.
				if !newEntries.Equals(currentEntries) {
					ch <- newEntries
				}
				currentEntries = newEntries
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return ch, nil
}

// Register is exported
func (s *Discovery) Register(addr string) error {
	return discovery.ErrNotImplemented
}
