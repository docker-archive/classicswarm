package cli

import (
	"flag"
	mcli "github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreateDiscovery(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	set.String("heartbeat", "12s", "s")
	globalSet := flag.NewFlagSet("test", 0)
	globalSet.Int("t", 1, "doc")
	c := mcli.NewContext(nil, set, globalSet)
	discovery := createDiscovery("token://6856663cdefdec325839a4b7e1de38e8 ", c)
	assert.NotNil(t, discovery)
}

func TestLoadTlsConfig(t *testing.T) {
	wd, _ := os.Getwd()
	ca := wd + "/test-tls/ca.pem"
	cert := wd + "/test-tls/cert.pem"
	key := wd + "/test-tls/key.pem"

	tlsConfig, err := loadTLSConfig(ca, cert, key, true)
	assert.Nil(t, err)
	assert.NotNil(t, tlsConfig)

	tlsConfigTwo, errTwo := loadTLSConfig(ca, cert, key, false)
	assert.Nil(t, errTwo)
	assert.NotNil(t, tlsConfigTwo)
}
