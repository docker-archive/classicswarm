// +build !windows

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatingEndpoint(t *testing.T) {
	assert.Equal(t, "/tmp/swarm-strategy-test.sock", createEndpoint("test"))
}
