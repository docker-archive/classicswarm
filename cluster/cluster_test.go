package cluster

import (
	"io/ioutil"
	"testing"

	"github.com/docker/swarm/state"
	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createNode(t *testing.T, ID string, containers ...dockerclient.Container) *Node {
	node := NewNode(ID, 0)
	node.Name = ID

	assert.False(t, node.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("ListContainers", true, false, "").Return(containers, nil)
	client.On("InspectContainer", mock.Anything).Return(
		&dockerclient.ContainerInfo{
			Config: &dockerclient.ContainerConfig{CpuShares: 100},
		}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything).Return()

	assert.NoError(t, node.connectClient(client))
	assert.True(t, node.IsConnected())
	node.ID = ID

	return node
}

func newCluster(t *testing.T) *Cluster {
	dir, err := ioutil.TempDir("", "store-test")
	assert.NoError(t, err)
	return NewCluster(state.NewStore(dir), nil, 0)
}

func TestAddNode(t *testing.T) {
	c := newCluster(t)
	assert.Equal(t, len(c.Nodes()), 0)
	assert.Nil(t, c.Node("test"))
	assert.Nil(t, c.Node("test2"))

	assert.NoError(t, c.AddNode(createNode(t, "test")))
	assert.Equal(t, len(c.Nodes()), 1)
	assert.NotNil(t, c.Node("test"))

	assert.Error(t, c.AddNode(createNode(t, "test")))
	assert.Equal(t, len(c.Nodes()), 1)
	assert.NotNil(t, c.Node("test"))

	assert.NoError(t, c.AddNode(createNode(t, "test2")))
	assert.Equal(t, len(c.Nodes()), 2)
	assert.NotNil(t, c.Node("test2"))
}

func TestContainerLookup(t *testing.T) {
	c := newCluster(t)
	container := dockerclient.Container{
		Id:    "container-id",
		Names: []string{"/container-name1", "/container-name2"},
	}
	node := createNode(t, "test-node", container)
	assert.NoError(t, c.AddNode(node))

	// Invalid lookup
	assert.Nil(t, c.Container("invalid-id"))
	assert.Nil(t, c.Container(""))
	// Container ID lookup.
	assert.NotNil(t, c.Container("container-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, c.Container("container-"))
	// Container name lookup.
	assert.NotNil(t, c.Container("container-name1"))
	assert.NotNil(t, c.Container("container-name2"))
	// Container node/name matching.
	assert.NotNil(t, c.Container("test-node/container-name1"))
	assert.NotNil(t, c.Container("test-node/container-name2"))
}

func TestDeployContainer(t *testing.T) {
	// Create a test node.
	node := createNode(t, "test")

	// Create a test cluster.
	c := newCluster(t)
	assert.NoError(t, c.AddNode(node))

	// Fake dockerclient calls to deploy a container.
	client := node.client.(*mockclient.MockClient)
	client.On("CreateContainer", mock.Anything, mock.Anything).Return("id", nil).Once()
	client.On("ListContainers", true, false, mock.Anything).Return([]dockerclient.Container{{Id: "id"}}, nil).Once()
	client.On("InspectContainer", "id").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()

	// Ensure the container gets deployed.
	container, err := c.DeployContainer(node, &dockerclient.ContainerConfig{}, "name")
	assert.NoError(t, err)
	assert.Equal(t, container.Id, "id")
}
