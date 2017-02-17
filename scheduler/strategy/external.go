package strategy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	schema "github.com/docker/swarm/scheduler/simplified-schema"
)

var (
	ErrURLMissing        = "External scheduler requires a url"
	ErrInvalidTimeoutOpt = "Invalid timeout value for external scheduler: %s"
	ErrInvalidRetriesOpt = "Invalid retry value for external scheduler: %s"
	ErrInvalidMarshalOpt = "Invalid marshal_cluster_state value for external scheduler: %s"

	ErrTimeout        = "External scheduler timed out while waiting for a response"
	ErrInvalidNodeID  = "External scheduler returned invalid node ID: %s"
	ErrMarshalError   = "Could not marshal placement for external scheduler: %s"
	ErrHTTPRequest    = "Error making HTTP request to external scheduler : %s"
	ErrHTTPResponse   = "Error reading response from external scheduler: %s"
	ErrJSONParseError = "Error parsing JSON from external scheduler: %s"
	ErrExternalError  = "External scheduler returned an error: %s"
)

type externalSchedulerResult struct {
	NodeIds []string `json:"nodes"`
	Error   string   `json:"error"`
}

// ExternalPlacementStrategy uses an external service to make the placement decision
type ExternalPlacementStrategy struct {
	url          string
	marshalState bool
	timeout      time.Duration
	retries      int
	client       *http.Client
}

// Initialize an ExternalPlacementStrategy.
func (p *ExternalPlacementStrategy) Initialize(opts map[string]string) error {
	if opts["url"] == "" {
		return errors.New(ErrURLMissing)
	}

	p.url = opts["url"]
	p.client = &http.Client{}
	p.marshalState = true
	p.timeout = time.Duration(3000 * time.Millisecond)
	p.retries = 3

	if opts["marshal_cluster_state"] != "" {
		marshal, err := strconv.ParseBool(opts["marshal_cluster_state"])
		if err != nil {
			return fmt.Errorf(ErrInvalidMarshalOpt, opts["marshal_cluster_state"])
		}
		p.marshalState = marshal
	}

	if opts["timeout"] != "" {
		timeout, err := strconv.Atoi(opts["timeout"])
		if err != nil {
			return fmt.Errorf(ErrInvalidTimeoutOpt, opts["timeout"])
		}

		if timeout < 1 {
			timeout = 1
		}

		p.timeout = time.Duration(timeout) * time.Millisecond
	}

	if opts["retries"] != "" {
		retries, err := strconv.Atoi(opts["retries"])
		if err != nil {
			return fmt.Errorf(ErrInvalidRetriesOpt, opts["retries"])
		}

		if retries < 0 {
			retries = 0
		}

		p.retries = retries
	}

	return nil
}

// Name returns the name of the strategy.
func (p *ExternalPlacementStrategy) Name() string {
	return "external"
}

// RankAndSort marshals config and node data to json, posts them to the http url and unmarshals the json response
func (p *ExternalPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	nodeMap := make(map[string]*node.Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var placement *schema.Placement
	if p.marshalState {
		placement = schema.SimplifyPlacement(config, nodes)
	} else {
		placement = schema.SimplifyPlacement(config, make([]*node.Node, 0))
	}

	data, err := json.Marshal(placement)
	if err != nil {
		return nil, fmt.Errorf(ErrMarshalError, err)
	}

	var errors []error
	for len(errors) <= p.retries {
		response, err := p.withTimeout(p.httpScheduler, &data)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		output, err := p.processResponse(response, nodeMap)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		return output, nil
	}

	for _, e := range errors {
		log.WithField("error", e).Debugf("External scheduler error during placement")
	}

	return nil, errors[len(errors)-1]
}

// httpScheduler makes an actual request over HTTP and handles the response body
func (p *ExternalPlacementStrategy) httpScheduler(data *[]byte) (io.Reader, error) {
	response, err := p.client.Post(p.url, "application/json", bytes.NewReader(*data))

	if err != nil {
		return nil, fmt.Errorf(ErrHTTPRequest, err)
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(ErrHTTPResponse, err)
	}

	return bytes.NewReader(body), nil
}

// withTimeout wraps the scheduler function (f) with timeout logic
func (p *ExternalPlacementStrategy) withTimeout(f func(data *[]byte) (io.Reader, error), data *[]byte) (io.Reader, error) {
	var extErr error
	result := make(chan io.Reader, 1)
	go func() {
		r, err := f(data)
		extErr = err
		result <- r
	}()

	select {
	case ret := <-result:
		return ret, extErr
	case <-time.After(p.timeout):
		return nil, errors.New(ErrTimeout)
	}
}

// processResponse decodes an external scheduler JSON response and turns it back in to a node list
func (p *ExternalPlacementStrategy) processResponse(response io.Reader, nodeMap map[string]*node.Node) ([]*node.Node, error) {
	var result externalSchedulerResult

	err := json.NewDecoder(response).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf(ErrJSONParseError, err)
	}

	if len(result.Error) > 0 {
		return nil, fmt.Errorf(ErrExternalError, result.Error)
	}

	output := make([]*node.Node, len(result.NodeIds))
	for i, n := range result.NodeIds {
		if _, ok := nodeMap[n]; !ok {
			return nil, fmt.Errorf(ErrInvalidNodeID, n)
		}
		output[i] = nodeMap[n]
	}

	return output, nil
}
