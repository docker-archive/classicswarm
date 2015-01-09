package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialise(t *testing.T) {
	discovery := &ListDiscoveryService{}
	discovery.Initialize("1.1.1.1:1111,2.2.2.2:2222", 0)
	assert.Equal(t, len(discovery.list), 2)
	assert.Equal(t, discovery.list[0].String(), "http://1.1.1.1:1111")
	assert.Equal(t, discovery.list[1].String(), "http://2.2.2.2:2222")
}

func TestRegister(t *testing.T) {
	discovery := &ListDiscoveryService{}
	assert.Error(t, discovery.Register("0.0.0.0"))
}
