package kubernetes

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/docker/engine-api/types"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/client/unversioned/testclient"
)

func TestNewCluster(t *testing.T) {
	client := testclient.NewSimpleFake()
	_, err := NewCluster(nil, "", client, nil, nil)
	assert.Nil(t, err)
}

func TestCreateContainer(t *testing.T) {}
func TestRemoveContainer(t *testing.T) {}
func TestContainers(t *testing.T)      {}
func TestContainer(t *testing.T)       {}
func TestImage(t *testing.T)           {}
func TestImages(t *testing.T)          {}
func TestInfo(t *testing.T)            {}

type fakeDockerEngine struct {
	server *httptest.Server
	host   string
	port   string
}

func newDockerEngineFake() *fakeDockerEngine {
	mux := http.NewServeMux()
	mux.HandleFunc("/info/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(types.Info{NCPU: 2, MemTotal: 1024})
	})
	mux.HandleFunc("/version/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(types.Version{Version: "1.9.1"})
	})
	mux.HandleFunc("/v1.21/containers/json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{})
	})
	mux.HandleFunc("/v1.21/images/json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	ts := httptest.NewServer(mux)

	dockerURL, _ := url.Parse(ts.URL)
	dockerHost, dockerPort, _ := net.SplitHostPort(dockerURL.Host)

	return &fakeDockerEngine{
		server: ts,
		host:   dockerHost,
		port:   dockerPort,
	}
}

func TestTotalCpus(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, c.TotalCpus(), int64(0))

	fakeDocker := newDockerEngineFake()
	defer fakeDocker.server.Close()

	client = testclient.NewSimpleFake(&api.NodeList{Items: []api.Node{
		{
			ObjectMeta: api.ObjectMeta{Name: "node1"},
			Status: api.NodeStatus{
				Capacity: api.ResourceList{
					api.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
					api.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
					api.ResourcePods:   *resource.NewQuantity(0, resource.DecimalSI),
				},
				Addresses: []api.NodeAddress{
					{
						Type:    api.NodeInternalIP,
						Address: fakeDocker.host,
					},
				},
			},
		},
	}})

	opts := cluster.DriverOpts([]string{fmt.Sprintf("docker.engineport=%s", fakeDocker.port)})
	c, err = NewCluster(nil, "", client, opts, &cluster.EngineOpts{})
	assert.Nil(t, err)
	assert.Equal(t, c.TotalCpus(), int64(2))
}

func TestTotalMemory(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, &cluster.EngineOpts{})
	assert.Nil(t, err)
	assert.Equal(t, c.TotalMemory(), int64(0))

	fakeDocker := newDockerEngineFake()
	defer fakeDocker.server.Close()

	client = testclient.NewSimpleFake(&api.NodeList{Items: []api.Node{
		{
			ObjectMeta: api.ObjectMeta{Name: "node1"},
			Status: api.NodeStatus{
				Capacity: api.ResourceList{
					api.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
					api.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
					api.ResourcePods:   *resource.NewQuantity(0, resource.DecimalSI),
				},
				Addresses: []api.NodeAddress{
					{
						Type:    api.NodeInternalIP,
						Address: fakeDocker.host,
					},
				},
			},
		},
	}})

	opts := cluster.DriverOpts([]string{fmt.Sprintf("docker.engineport=%s", fakeDocker.port)})
	c, err = NewCluster(nil, "", client, opts, &cluster.EngineOpts{})
	assert.Nil(t, err)
	assert.Equal(t, c.TotalMemory(), int64(1024))
}

var formatContainerIDTests = []struct {
	in  string
	out string
}{
	{"", ""},
	{"docker://796f440485b2", "796f440485b2"},
	{"broken://796f440485b2", "broken://796f440485b2"},
}

func TestFormatContainerID(t *testing.T) {
	for _, tt := range formatContainerIDTests {
		assert.Equal(t, formatContainerID(tt.in), tt.out)
	}
}

func TestDockerEngines(t *testing.T) {
	// Test empty nodes.
	client := testclient.NewSimpleFake()
	engines, err := dockerEngines(client, &cluster.EngineOpts{}, defaultDockerEnginePort)
	assert.Nil(t, err)
	assert.Len(t, engines, 0, "The size of engines is not 0")

	// Test non-empty nodes.
	fakeEngine := newDockerEngineFake()
	defer fakeEngine.server.Close()

	client = testclient.NewSimpleFake(&api.NodeList{Items: []api.Node{
		{
			ObjectMeta: api.ObjectMeta{Name: "node1"},
			Status: api.NodeStatus{
				Addresses: []api.NodeAddress{
					{
						Type:    api.NodeInternalIP,
						Address: fakeEngine.host,
					},
				}},
		},
	}})

	engines, err = dockerEngines(client, &cluster.EngineOpts{}, fakeEngine.port)
	assert.Nil(t, err)
	assert.Len(t, engines, 1, "The size of engines is not 1")
}

func TestRemoveImages(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, &cluster.EngineOpts{})
	assert.Nil(t, err)

	_, err = c.RemoveImages("image-1", false)
	assert.EqualError(t, err, "not supported with kubernetes")
}

func TestCreateNetwork(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, &cluster.EngineOpts{})
	assert.Nil(t, err)

	_, err = c.CreateNetwork("", nil)
	assert.EqualError(t, err, "not supported with kubernetes")
}

func TestRemoveNetwork(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, &cluster.EngineOpts{})
	assert.Nil(t, err)

	assert.EqualError(t, c.RemoveNetwork(nil), "not supported with kubernetes")
}

func TestRenameContainer(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil, &cluster.EngineOpts{})
	assert.Nil(t, err)

	assert.EqualError(t, c.RenameContainer(nil, ""), "not supported with kubernetes")
}
