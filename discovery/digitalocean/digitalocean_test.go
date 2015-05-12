package digitalocean

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
