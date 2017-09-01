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

func (d *Discovery) SetAgentPort(discoveryEnginePort int) {
	d.agentPort = discoveryEnginePort
}

// Initialize initializes the discovery with the SwarmKit manager's address
func (d *Discovery) Initialize(uri string, heartbeat time.Duration, ttl time.Duration, _ map[string]string) error {
	// the uri may be a manager address or a docker socket
	if uri == "" {
		return errors.New("Swarm Mode manager address not provided")
	}
	d.uri = uri
	d.heartbeat = heartbeat
	d.ttl = ttl

	// create a client to connect with the Docker API
	httpClient, _, err := cluster.NewHTTPClientTimeout("tcp://"+uri, nil, time.Duration(30*time.Second), nil)
	if err != nil {
		return err
	}
	d.httpClient = httpClient

	apiClient, err := client.NewClient("tcp://"+uri, "", d.httpClient, nil)
	if err != nil {
		return err
	}
	d.apiClient = apiClient

	return nil
}

// fetch returns the list of entries for the discovery service at the specified endpoint.
func (d *Discovery) fetch() (discovery.Entries, error) {
	ctx := context.Background()
	nodeList, err := d.apiClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch entries, Discovery service returned %s", err.Error())
	}

	var addrs []string

	for _, n := range nodeList {
		// Each agent needs to expose the same port
		addrs = append(addrs, n.Status.Addr+":"+strconv.Itoa(d.agentPort))
	}

	return discovery.CreateEntries(addrs)
}

// Watch is exported.
func (d *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	ticker := time.NewTicker(d.heartbeat)
	errCh := make(chan error)

	// eventsOptions sets a filter to only retrieve cluster level events
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "node")
	eventsOptions := types.EventsOptions{
		Filters: eventFilter,
	}
	ctx, cancel := context.WithCancel(context.Background())
	eventsChan, eventsErrChan := d.apiClient.Events(ctx, eventsOptions)

	go func() {
		defer close(ch)
		defer close(errCh)
		defer cancel()

		// Send the initial entries if available.
		currentEntries, err := d.fetch()
		if err != nil {
			errCh <- err
		} else {
			ch <- currentEntries
		}

		// Periodically send updates, unless an event is received
		for {
			select {
			case <-ticker.C:
				newEntries, err := d.fetch()
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
				newEntries, err := d.fetch()
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
func (d *Discovery) Register(addr string) error {
	return nil
}
