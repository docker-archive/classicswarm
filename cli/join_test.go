package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAddrFormat(t *testing.T) {
	assert.False(t, checkAddrFormat("1.1.1.1"))
	assert.False(t, checkAddrFormat("hostname"))
	assert.False(t, checkAddrFormat("1.1.1.1:"))
	assert.False(t, checkAddrFormat("hostname:"))
	assert.False(t, checkAddrFormat("1.1.1.1:111111"))
	assert.False(t, checkAddrFormat("hostname:111111"))
	assert.False(t, checkAddrFormat("http://1.1.1.1"))
	assert.False(t, checkAddrFormat("http://hostname"))
	assert.False(t, checkAddrFormat("http://1.1.1.1:1"))
	assert.False(t, checkAddrFormat("http://hostname:1"))
	assert.False(t, checkAddrFormat(":1.1.1.1"))
	assert.False(t, checkAddrFormat(":hostname"))
	assert.False(t, checkAddrFormat(":1.1.1.1:1"))
	assert.False(t, checkAddrFormat(":hostname:1"))
	assert.True(t, checkAddrFormat("1.1.1.1:1111"))
	assert.True(t, checkAddrFormat("hostname:1111"))
	assert.True(t, checkAddrFormat("host-name_42:1111"))
	assert.False(t, checkAddrFormat("1.1.1.1:-1"))
	assert.False(t, checkAddrFormat("1.1.1.1:0"))
	assert.True(t, checkAddrFormat("1.1.1.1:65535"))
	assert.False(t, checkAddrFormat("1.1.1.1:65536"))
	assert.False(t, checkAddrFormat("1.1.1.1: 4000"))
	assert.False(t, checkAddrFormat("1.1.1.1:m2"))
	assert.True(t, checkAddrFormat("[2001:db8:0:f101::3]:2375"))
	assert.False(t, checkAddrFormat("2001:db8:0:f101::3:2375"))
	assert.False(t, checkAddrFormat("[2001:db8:0:f101::3]:3:2375"))
}
