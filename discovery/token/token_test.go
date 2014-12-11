package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	discovery, _ := Init("token")
	if dtoken, ok := discovery.(TokenDiscoveryService); ok {
		assert.Equal(t, dtoken.token, "token")
		assert.Equal(t, dtoken.url, DISCOVERY_URL)
	}

	discovery, _ = Init("custom/path/token")
	if dtoken, ok := discovery.(TokenDiscoveryService); ok {
		assert.Equal(t, dtoken.token, "token")
		assert.Equal(t, dtoken.url, "https://custom/path")
	}
}

func TestRegister(t *testing.T) {
	discovery := TokenDiscoveryService{token: "TEST_TOKEN", url: DISCOVERY_URL}
	expected := "127.0.0.1:2675"
	assert.NoError(t, discovery.Register(expected))

	addrs, err := discovery.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, len(addrs), 1)
	assert.Equal(t, addrs[0].String(), "http://"+expected)

	assert.NoError(t, discovery.Register(expected))
}
