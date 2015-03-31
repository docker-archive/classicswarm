package digitalocean

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &DigitalOceanService{}
	discovery.Initialize("digitalocean_token", 0)
	assert.Equal(t, discovery.token, "digitalocean_token")
}

func TestRegister(t *testing.T) {
	discovery := &DigitalOceanService{token: "token"}
	assert.Error(t, discovery.Register("0.0.0.0"))
}