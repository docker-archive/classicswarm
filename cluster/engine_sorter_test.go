package cluster

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeSorter(t *testing.T) {
	nodes := []*Engine{{Name: "name1"}, {Name: "name3"}, {Name: "name2"}}

	sort.Sort(EngineSorter(nodes))

	assert.Equal(t, nodes[0].Name, "name1")
	assert.Equal(t, nodes[1].Name, "name2")
	assert.Equal(t, nodes[2].Name, "name3")
}
