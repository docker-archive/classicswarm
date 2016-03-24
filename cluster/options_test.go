package cluster

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var opts = DriverOpts{
	"foo1=bar",
	"foo2=-5",
	"foo3=-5ss",
	"foo4=7",
	"foo5=7ss",
	"foo6=0.6",
	"foo7=0.6ss",
	"foo8=127.0.0.1",
	"foo9=127.0.0.1oo",
	"foo10=true",
	"foo11=truess",
}

func TestString(t *testing.T) {
	if err := os.Setenv("FOO_12", "bar"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.String("foo1", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "bar")

	val, err = opts.String("foo2", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "-5")

	val, err = opts.String("foo3", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "-5ss")

	val, err = opts.String("foo4", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "7")

	val, err = opts.String("foo5", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "7ss")

	val, err = opts.String("foo6", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "0.6")

	val, err = opts.String("foo7", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "0.6ss")

	val, err = opts.String("foo8", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "127.0.0.1")

	val, err = opts.String("foo9", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "127.0.0.1oo")

	val, err = opts.String("foo10", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "true")

	val, err = opts.String("foo11", "")
	assert.Nil(t, err)
	assert.Equal(t, val, "truess")

	val, err = opts.String("", "FOO_12")
	assert.Nil(t, err)
	assert.Equal(t, val, "bar")

	val, err = opts.String("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, "")
}

func TestInt(t *testing.T) {
	if err := os.Setenv("FOO_12", "bar"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.Int("foo1", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo2", "")
	assert.Nil(t, err)
	assert.Equal(t, val, int64(-5))

	val, err = opts.Int("foo3", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo4", "")
	assert.Nil(t, err)
	assert.Equal(t, val, int64(7))

	val, err = opts.Int("foo5", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo6", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo7", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo8", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo9", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo10", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("foo11", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("", "FOO_12")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))

	val, err = opts.Int("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, int64(0))
}

func TestUint(t *testing.T) {
	if err := os.Setenv("FOO_12", "bar"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.Uint("foo1", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo2", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo3", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo4", "")
	assert.Nil(t, err)
	assert.Equal(t, val, uint64(7))

	val, err = opts.Uint("foo5", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo6", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo7", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo8", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo9", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo10", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("foo11", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("", "FOO_12")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))

	val, err = opts.Uint("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, uint64(0))
}

func TestFloat(t *testing.T) {
	if err := os.Setenv("FOO_12", "0.2"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.Float("foo1", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo2", "")
	assert.Nil(t, err)
	assert.Equal(t, val, -5.0)

	val, err = opts.Float("foo3", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo4", "")
	assert.Nil(t, err)
	assert.Equal(t, val, 7.0)

	val, err = opts.Float("foo5", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo6", "")
	assert.Nil(t, err)
	assert.Equal(t, val, 0.6)

	val, err = opts.Float("foo7", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo8", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo9", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo10", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("foo11", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)

	val, err = opts.Float("", "FOO_12")
	assert.Nil(t, err)
	assert.Equal(t, val, 0.2)

	val, err = opts.Float("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, 0.0)
}

func TestIP(t *testing.T) {
	if err := os.Setenv("FOO_12", "0.0.0.0"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.IP("foo1", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo2", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo3", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo4", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo5", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo6", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo7", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo8", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP("127.0.0.1"))

	val, err = opts.IP("foo9", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo10", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("foo11", "")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP(""))

	val, err = opts.IP("", "FOO_12")
	assert.Nil(t, err)
	assert.Equal(t, val, net.ParseIP("0.0.0.0"))

	val, err = opts.IP("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, net.ParseIP(""))
}

func TestBool(t *testing.T) {
	if err := os.Setenv("FOO_12", "true"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("FOO_12")

	val, err := opts.Bool("foo1", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo2", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo3", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo4", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo5", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo6", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo7", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo8", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo9", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("foo10", "")
	assert.Nil(t, err)
	assert.Equal(t, val, true)

	val, err = opts.Bool("foo11", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)

	val, err = opts.Bool("", "FOO_12")
	assert.Nil(t, err)
	assert.Equal(t, val, true)

	val, err = opts.Bool("invalid", "")
	assert.NotNil(t, err)
	assert.Equal(t, val, false)
}
