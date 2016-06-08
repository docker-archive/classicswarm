package strategy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/swarm/scheduler/node"
	schema "github.com/docker/swarm/scheduler/simplified-schema"
	"github.com/stretchr/testify/assert"
)

func mockExternalServer(scheduler func(req *schema.Placement) string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := &schema.Placement{}
		data, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(data, p)
		fmt.Fprintf(w, scheduler(p))
	}))
}

func mockSchedulerFirstNode(req *schema.Placement) string {
	return fmt.Sprintf("[\"%s\"]", req.Nodes[0].ID)
}

func mockSchedulerTimeout(req *schema.Placement) string {
	time.Sleep(20 * time.Millisecond)
	return mockSchedulerFirstNode(req)
}

func mockSchedulerWithRetryFailures(targetCount int, failResult string, scheduler func(*schema.Placement) string) func(*schema.Placement) string {
	count := 0
	return func(req *schema.Placement) string {
		if count >= targetCount {
			return scheduler(req)
		} else {
			count++
			return failResult
		}
	}
}

func setupExternalPlacementStrategy(url string, timeout int, retries int) (*ExternalPlacementStrategy, []*node.Node) {
	opts := make(map[string]string)
	opts["url"] = url
	opts["retries"] = fmt.Sprintf("%d", retries)
	opts["timeout"] = fmt.Sprintf("%d", timeout)

	s := &ExternalPlacementStrategy{}
	s.Initialize(opts)

	nodes := []*node.Node{
		createNode(fmt.Sprintf("node-0"), 1, 1),
	}

	return s, nodes
}

func TestExternalPlaceSimple(t *testing.T) {
	server := mockExternalServer(mockSchedulerFirstNode)
	defer server.Close()

	s, nodes := setupExternalPlacementStrategy(server.URL, 3, 3) // url, timeout(ms), retries

	// add 10 containers
	for i := 0; i < 10; i++ {
		config := createConfig(0, 0)
		node := selectTopNode(t, s, config, nodes)
		assert.NoError(t, node.AddContainer(createContainer(fmt.Sprintf("c%d", i), config)))
	}

	assert.Equal(t, 10, len(nodes[0].Containers))
}

func TestExternalPlaceRetries(t *testing.T) {
	server := mockExternalServer(
		mockSchedulerWithRetryFailures(1, "[]", mockSchedulerFirstNode),
	)
	defer server.Close()

	s, nodes := setupExternalPlacementStrategy(server.URL, 10, 2) // url, timeout(ms), retries

	config := createConfig(0, 0)
	_, err := s.RankAndSort(config, nodes)

	assert.NoError(t, err)
}

func TestExternalErrorTimeout(t *testing.T) {
	server := mockExternalServer(mockSchedulerTimeout)
	defer server.Close()

	s, nodes := setupExternalPlacementStrategy(server.URL, 10, 0) // url, timeout(ms), retries

	config := createConfig(0, 0)
	_, err := s.RankAndSort(config, nodes)

	assert.EqualError(t, err, "External scheduler timed out while waiting for a response")
}

func TestExternalErrorBadJson(t *testing.T) {
	server := mockExternalServer(func(req *schema.Placement) string { return "{" })
	defer server.Close()

	s, nodes := setupExternalPlacementStrategy(server.URL, 3000, 0) // url, timeout(ms), retries

	config := createConfig(0, 0)
	_, err := s.RankAndSort(config, nodes)

	assert.EqualError(t, err, "Error parsing JSON from external scheduler: unexpected EOF")
}

func TestExternalErrorBadNodeID(t *testing.T) {
	server := mockExternalServer(func(req *schema.Placement) string { return "[\"bad-node\"]" })
	defer server.Close()

	s, nodes := setupExternalPlacementStrategy(server.URL, 3000, 0) // url, timeout(ms), retries

	config := createConfig(0, 0)
	_, err := s.RankAndSort(config, nodes)

	assert.EqualError(t, err, "External scheduler returned invalid node ID: bad-node")
}

func TestExternalInitializeErrors(t *testing.T) {
	opts1 := make(map[string]string)
	s1 := &ExternalPlacementStrategy{}
	err1 := s1.Initialize(opts1)
	assert.EqualError(t, err1, "External scheduler requires a url")

	opts2 := make(map[string]string)
	opts2["url"] = "http://"
	opts2["timeout"] = "badval"
	s2 := &ExternalPlacementStrategy{}
	err2 := s2.Initialize(opts2)
	assert.EqualError(t, err2, "Invalid timeout value for external scheduler: badval")

	opts3 := make(map[string]string)
	opts3["url"] = "http://"
	opts3["retries"] = "badval"
	s3 := &ExternalPlacementStrategy{}
	err3 := s3.Initialize(opts3)
	assert.EqualError(t, err3, "Invalid retry value for external scheduler: badval")

	opts4 := make(map[string]string)
	opts4["url"] = "http://"
	opts4["marshal_cluster_state"] = "badval"
	s4 := &ExternalPlacementStrategy{}
	err4 := s4.Initialize(opts4)
	assert.EqualError(t, err4, "Invalid marshal_cluster_state value for external scheduler: badval")
}
