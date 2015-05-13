package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var opts = DriverOpts{"foo1=bar", "foo2=-5", "foo3=7", "foo4=0.6"}

func TestString(t *testing.T) {
	val, ok := opts.String("foo1")
	assert.True(t, ok)
	assert.Equal(t, val, "bar")

	val, ok = opts.String("foo2")
	assert.True(t, ok)
	assert.Equal(t, val, "-5")

	val, ok = opts.String("foo3")
	assert.True(t, ok)
	assert.Equal(t, val, "7")

	val, ok = opts.String("foo4")
	assert.True(t, ok)
	assert.Equal(t, val, "0.6")

	val, ok = opts.String("invalid")
	assert.False(t, ok)
	assert.Equal(t, val, "")
}

func TestInt(t *testing.T) {
	val, ok := opts.Int("foo1")
	assert.True(t, ok)
	assert.Equal(t, val, 0)

	val, ok = opts.Int("foo2")
	assert.True(t, ok)
	assert.Equal(t, val, -5)

	val, ok = opts.Int("foo3")
	assert.True(t, ok)
	assert.Equal(t, val, 7)

	val, ok = opts.Int("foo4")
	assert.True(t, ok)
	assert.Equal(t, val, 0)

	val, ok = opts.Int("invalid")
	assert.False(t, ok)
	assert.Equal(t, val, 0)
}

func TestUint(t *testing.T) {
	val, ok := opts.Uint("foo1")
	assert.True(t, ok)
	assert.Equal(t, val, uint(0))

	val, ok = opts.Uint("foo2")
	assert.True(t, ok)
	assert.Equal(t, val, uint(0))

	val, ok = opts.Uint("foo3")
	assert.True(t, ok)
	assert.Equal(t, val, uint(7))

	val, ok = opts.Uint("foo4")
	assert.True(t, ok)
	assert.Equal(t, val, uint(0))

	val, ok = opts.Uint("invalid")
	assert.False(t, ok)
	assert.Equal(t, val, uint(0))
}

func TestFloat(t *testing.T) {
	val, ok := opts.Float("foo1")
	assert.True(t, ok)
	assert.Equal(t, val, 0.0)

	val, ok = opts.Float("foo2")
	assert.True(t, ok)
	assert.Equal(t, val, -5.0)

	val, ok = opts.Float("foo3")
	assert.True(t, ok)
	assert.Equal(t, val, 7.0)

	val, ok = opts.Float("foo4")
	assert.True(t, ok)
	assert.Equal(t, val, 0.6)

	val, ok = opts.Float("invalid")
	assert.False(t, ok)
	assert.Equal(t, val, 0.0)
}
