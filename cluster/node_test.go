package cluster

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeSorter(t *testing.T) {
	nodes := []Node{&FakeNode{"name1"}, &FakeNode{"name3"}, &FakeNode{"name2"}}

	sort.Sort(NodeSorter(nodes))

	assert.Equal(t, nodes[0].Name(), "name1")
	assert.Equal(t, nodes[1].Name(), "name2")
	assert.Equal(t, nodes[2].Name(), "name3")
}
