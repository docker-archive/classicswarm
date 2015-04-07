package cluster

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineSorter(t *testing.T) {
	engines := []*Engine{{Name: "name1"}, {Name: "name3"}, {Name: "name2"}}

	sort.Sort(EngineSorter(engines))

	assert.Equal(t, engines[0].Name, "name1")
	assert.Equal(t, engines[1].Name, "name2")
	assert.Equal(t, engines[2].Name, "name3")
}
