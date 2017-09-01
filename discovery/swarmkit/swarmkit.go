package swarmkit

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/discovery"
	"github.com/docker/swarm/cluster"
)

const defaultAgentPort = "2375"

// Discovery is exported.
type Discovery struct {
	heartbeat  time.Duration
	ttl        time.Duration
	httpClient *http.Client
	apiClient  *client.Client
	uri        string
	token      string
	agentPort  int
}

func init() {
	Init()
}

// Init is exported.
func Init() {
	discovery.Register("swarmkit", &Discovery{})
}

func (s *Discovery) SetAgentPort(discoveryEnginePort int) {
	s.agentPort = discoveryEnginePort
}

// Initialize initializes the discovery with the SwarmKit manager's address
func (s *Discovery) Initialize(uri string, heartbeat time.Duration, ttl time.Duration, _ map[string]string) error {
	// the uri may be a manager address or a docker socket
	if uri == "" {
		return errors.New("Swarm Mode manager address not provided")
	}
	s.uri = uri
	s.heartbeat = heartbeat
	s.ttl = ttl

	// create a client to connect with the Docker API
	httpClient, _, err := cluster.NewHTTPClientTimeout("tcp://"+uri, nil, time.Duration(30*time.Second), nil)
	if err != nil {
		return err
	}
	s.httpClient = httpClient

	apiClient, err := client.NewClient("tcp://"+uri, "", s.httpClient, nil)
	if err != nil {
		return err
	}
	s.apiClient = apiClient

	return nil
}

// fetch returns the list of entries for the discovery service at the specified endpoint.
func (s *Discovery) fetch() (discovery.Entries, error) {
	ctx := context.Background()
	nodeList, err := s.apiClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch entries, Discovery service returned %s", err.Error())
	}

	var addrs []string

	for _, n := range nodeList {
		// Each agent needs to expose the same port
		addrs = append(addrs, n.Status.Addr+":"+strconv.Itoa(s.agentPort))
	}

	return discovery.CreateEntries(addrs)
}

// Watch is exported.
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	ticker := time.NewTicker(s.heartbeat)
	errCh := make(chan error)

	// eventsOptions sets a filter to only retrieve cluster level events
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "node")
	eventsOptions := types.EventsOptions{
		Filters: eventFilter,
	}
	ctx, cancel := context.WithCancel(context.Background())
	eventsChan, eventsErrChan := s.apiClient.Events(ctx, eventsOptions)

	go func() {
		defer close(ch)
		defer close(errCh)
		defer cancel()

		// Send the initial entries if available.
		currentEntries, err := s.fetch()
		if err != nil {
			errCh <- err
		} else {
			ch <- currentEntries
		}

		// Periodically send updates, unless an event is received
		for {
			select {
			case <-ticker.C:
				newEntries, err := s.fetch()
				if err != nil {
					errCh <- err
					continue
				}

				// Check if the list of nodes has really changed.
				if !newEntries.Equals(currentEntries) {
					ch <- newEntries
				}
				currentEntries = newEntries
			case <-eventsChan:
				newEntries, err := s.fetch()
				if err != nil {
					errCh <- err
					continue
				}

				// Check if the list of nodes has really changed.
				if !newEntries.Equals(currentEntries) {
					ch <- newEntries
				}
				currentEntries = newEntries
			case err := <-eventsErrChan:
				errCh <- err
				continue
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return ch, errCh
}

// Register does nothing because Swarm Mode manages the node inventory
func (s *Discovery) Register(addr string) error {
	return nil
}
