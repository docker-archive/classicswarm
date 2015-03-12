package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &FileDiscoveryService{}
	discovery.Initialize("/path/to/file", 0)
	assert.Equal(t, discovery.path, "/path/to/file")
}

func TestContent(t *testing.T) {
	data := `
1.1.1.[1:2]:1111
2.2.2.[2:4]:2222
`
	ips := parseFileContent([]byte(data))
	assert.Equal(t, ips[0], "1.1.1.1:1111")
	assert.Equal(t, ips[1], "1.1.1.2:1111")
	assert.Equal(t, ips[2], "2.2.2.2:2222")
	assert.Equal(t, ips[3], "2.2.2.3:2222")
	assert.Equal(t, ips[4], "2.2.2.4:2222")
}

func TestRegister(t *testing.T) {
	discovery := &FileDiscoveryService{path: "/path/to/file"}
	assert.Error(t, discovery.Register("0.0.0.0"))
}
