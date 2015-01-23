package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	scheme, uri := parse("127.0.0.1")
	assert.Equal(t, scheme, "nodes")
	assert.Equal(t, uri, "127.0.0.1")

	scheme, uri = parse("localhost")
	assert.Equal(t, scheme, "nodes")
	assert.Equal(t, uri, "localhost")

	scheme, uri = parse("scheme://127.0.0.1")
	assert.Equal(t, scheme, "scheme")
	assert.Equal(t, uri, "127.0.0.1")

	scheme, uri = parse("scheme://localhost")
	assert.Equal(t, scheme, "scheme")
	assert.Equal(t, uri, "localhost")

	scheme, uri = parse("")
	assert.Equal(t, scheme, "nodes")
	assert.Equal(t, uri, "")
}
