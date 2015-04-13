package mesos

import (
	"sort"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func TestSlaveSorter(t *testing.T) {
	slaves := []*slave{{cluster.Engine{Name: "name1"}, nil, nil, nil},
		{cluster.Engine{Name: "name2"}, nil, nil, nil},
		{cluster.Engine{Name: "name3"}, nil, nil, nil}}

	sort.Sort(SlaveSorter(slaves))

	assert.Equal(t, slaves[0].Name, "name1")
	assert.Equal(t, slaves[1].Name, "name2")
	assert.Equal(t, slaves[2].Name, "name3")
}
