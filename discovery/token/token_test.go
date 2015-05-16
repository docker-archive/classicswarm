package token

import (
	"log"
	"testing"
	"time"

	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &Discovery{}
	err := discovery.Initialize("token", 0)
	assert.NoError(t, err)
	assert.Equal(t, discovery.token, "token")
	assert.Equal(t, discovery.url, DiscoveryURL)

	err = discovery.Initialize("custom/path/token", 0)
	assert.NoError(t, err)
	assert.Equal(t, discovery.token, "token")
	assert.Equal(t, discovery.url, "https://custom/path")

	err = discovery.Initialize("", 0)
	assert.Error(t, err)
}

func TestRegister(t *testing.T) {
	d := &Discovery{token: "TEST_TOKEN", url: DiscoveryURL, heartbeat: 1}
	expected := "127.0.0.1:2675"
	expectedEntries, err := discovery.CreateEntries([]string{expected})
	assert.NoError(t, err)

	// Register
	assert.NoError(t, d.Register(expected))

	// Watch
	ch, err := d.Watch(nil)
	assert.NoError(t, err)
	select {
	case entries := <-ch:
		log.Printf("%v %v", entries, expectedEntries)
		assert.True(t, entries.Equals(expectedEntries))
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out")
	}

	assert.NoError(t, d.Register(expected))
}
