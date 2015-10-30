package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned/testclient"
)

func TestNewCluster(t *testing.T) {
	client := testclient.NewSimpleFake()
	_, err := NewCluster(nil, "", client, nil)
	assert.Nil(t, err)
}

func TestCreateContainer(t *testing.T) {}
func TestRemoveContainer(t *testing.T) {}
func TestContainers(t *testing.T)      {}
func TestContainer(t *testing.T)       {}
func TestImage(t *testing.T)           {}
func TestImages(t *testing.T)          {}
func TestInfo(t *testing.T)            {}
func TestTotalCpus(t *testing.T)       {}
func TestTotalMemory(t *testing.T)     {}

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
	engines, err := dockerEngines(client)
	assert.Nil(t, err)
	assert.Len(t, engines, 0, "The size of engines is not 0")

	// Test non-empty nodes.
	client = testclient.NewSimpleFake(&api.NodeList{Items: []api.Node{
		{ObjectMeta: api.ObjectMeta{Name: "node1"}}}})
	engines, err = dockerEngines(client)
	assert.Nil(t, err)
	assert.Len(t, engines, 1, "The size of engines is not 1")
}

func TestRemoveImages(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil)
	assert.Nil(t, err)

	_, err = c.RemoveImages("image-1", false)
	assert.EqualError(t, err, "not supported with kubernetes")
}

func TestCreateNetwork(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil)
	assert.Nil(t, err)

	_, err = c.CreateNetwork(nil)
	assert.EqualError(t, err, "not supported with kubernetes")
}

func TestRemoveNetwork(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil)
	assert.Nil(t, err)

	assert.EqualError(t, c.RemoveNetwork(nil), "not supported with kubernetes")
}

func TestRenameContainer(t *testing.T) {
	client := testclient.NewSimpleFake()
	c, err := NewCluster(nil, "", client, nil)
	assert.Nil(t, err)

	assert.EqualError(t, c.RenameContainer(nil, ""), "not supported with kubernetes")
}
