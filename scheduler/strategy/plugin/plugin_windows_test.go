// +build windows

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatingEndpoint(t *testing.T) {
	assert.Equal(t, `\\.\pipe\swarm-strategy-test`, createEndpoint("test"))
}
