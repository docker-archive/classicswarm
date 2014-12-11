package cluster

import (
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createNode(t *testing.T, ID string) *Node {
	node := NewNode(ID)

	assert.False(t, node.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything).Return()

	assert.NoError(t, node.connectClient(client))
	assert.True(t, node.IsConnected())
	node.ID = ID

	return node
}

func TestAddNode(t *testing.T) {
	c := NewCluster()

	assert.Equal(t, len(c.Nodes()), 0)

	assert.NoError(t, c.AddNode(createNode(t, "test")))
	assert.Equal(t, len(c.Nodes()), 1)

	assert.Error(t, c.AddNode(createNode(t, "test")))
	assert.Equal(t, len(c.Nodes()), 1)

	assert.NoError(t, c.AddNode(createNode(t, "test2")))
	assert.Equal(t, len(c.Nodes()), 2)
}
