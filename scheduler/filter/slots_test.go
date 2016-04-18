package filter

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/stretchr/testify/assert"
)

var labelsWithSlots = make(map[string]string)
var labelsWithoutSlots = make(map[string]string)
var labelsWithStringSlot = make(map[string]string)

func testFixturesAllFreeNode() []*node.Node {
	return []*node.Node{
		{
			ID:     "node-0-id",
			Name:   "node-0-name",
			Labels: labelsWithSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
			},
		},
		{
			ID:         "node-1-id",
			Name:       "node-1-name",
			Labels:     labelsWithSlots,
			Containers: []*cluster.Container{},
		},
	}
}

func testFixturesPartlyFreeNode() []*node.Node {
	return []*node.Node{
		{
			ID:     "node-0-id",
			Name:   "node-0-name",
			Labels: labelsWithSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},
		{
			ID:         "node-1-id",
			Name:       "node-1-name",
			Labels:     labelsWithSlots,
			Containers: []*cluster.Container{},
		},
	}
}

func testFixturesAllNoLabelNode() []*node.Node {
	return []*node.Node{
		{
			ID:     "node-0-id",
			Name:   "node-0-name",
			Labels: labelsWithoutSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},

		{
			ID:         "node-1-id",
			Name:       "node-1-name",
			Labels:     labelsWithoutSlots,
			Containers: []*cluster.Container{},
		},
	}
}

func testFixturesNoFreeNode() []*node.Node {
	return []*node.Node{
		{
			ID:     "node-0-id",
			Name:   "node-0-name",
			Labels: labelsWithSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},

		{
			ID:     "node-1-id",
			Name:   "node-1-name",
			Labels: labelsWithSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},
	}
}

func testFixturesNoFreeNodeButStringLabel() []*node.Node {
	return []*node.Node{
		{
			ID:     "node-0-id",
			Name:   "node-0-name",
			Labels: labelsWithSlots,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},

		{
			ID:     "node-1-id",
			Name:   "node-1-name",
			Labels: labelsWithStringSlot,
			Containers: []*cluster.Container{
				{Container: types.Container{}},
				{Container: types.Container{}},
				{Container: types.Container{}},
			},
		},
	}
}

func TestSlotsFilter(t *testing.T) {

	labelsWithSlots["containerslots"] = "3"
	labelsWithStringSlot["containerslots"] = "foo"

	var (
		f                         = SlotsFilter{}
		nodesAllFree              = testFixturesAllFreeNode()
		nodesPartlyFree           = testFixturesPartlyFreeNode()
		nodesAllNoLabel           = testFixturesAllNoLabelNode()
		nodesNoFree               = testFixturesNoFreeNode()
		nodesNoFreeButStringLabel = testFixturesNoFreeNodeButStringLabel()
		result                    []*node.Node
		err                       error
	)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesAllFree, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodesAllFree)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesPartlyFree, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodesPartlyFree[1])

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesAllNoLabel, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodesAllNoLabel)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesNoFree, true)
	assert.Equal(t, err, ErrNoNodeWithFreeSlotsAvailable)
	assert.Nil(t, result)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesNoFreeButStringLabel, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodesNoFreeButStringLabel[1])
}
