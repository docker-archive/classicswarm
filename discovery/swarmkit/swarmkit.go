package swarmkit

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/discovery"
	"github.com/docker/swarm/cluster"
)

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
	d.apiClient.NegotiateAPIVersion(context.Background())

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

		// eventConnRetries keeps track of how many retries have been made to
		// connect to the event stream
		eventConnRetries := 0

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
				log.Warnf("SwarmKit discovery: events stream failed with error: %s", err.Error())
				// if the event stream has failed over 5 times, then don't try to reconnect
				if eventConnRetries > 5 {
					errCh <- err
					return
				}
				// cancel context for the current event stream and create a new context
				// for a new connection
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				// if the events stream returns an error, try reconnecting. Discovery should not
				// fail if the events stream throws an error for any reason.
				// TODO(swarmkitdiscovery): We may want to have a maximum number of retries for
				// reconnecting to the events stream, possibly with a backoff mechanism.
				log.Debugf("SwarmKit discovery: creating a new events stream")
				eventsChan, eventsErrChan = d.apiClient.Events(ctx, eventsOptions)

				eventConnRetries++
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
