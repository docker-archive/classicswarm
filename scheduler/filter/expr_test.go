package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExprs(t *testing.T) {
	// Cannot use the leading digit for key
	_, err := parseExprs([]string{"1node"})
	assert.Error(t, err)

	// Cannot use space in key
	_, err = parseExprs([]string{"node ==node1"})
	assert.Error(t, err)

	// Cannot use * in key
	_, err = parseExprs([]string{"no*de==node1"})
	assert.Error(t, err)

	// Cannot use $ in key
	_, err = parseExprs([]string{"no$de==node1"})
	assert.Error(t, err)

	// Allow CAPS in key
	exprs, err := parseExprs([]string{"NoDe==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "NoDe")
	assert.Equal(t, exprs[0].value, "node1")

	// Allow dot in key
	exprs, err = parseExprs([]string{"no.de==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "no.de")
	assert.Equal(t, exprs[0].value, "node1")

	// Allow leading underscore
	exprs, err = parseExprs([]string{"_node==_node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "_node")
	assert.Equal(t, exprs[0].value, "_node1")

	// Allow globbing
	exprs, err = parseExprs([]string{"node==*node*"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "node")
	assert.Equal(t, exprs[0].value, "*node*")

	// Allow regexp in value
	exprs, err = parseExprs([]string{"node==/(?i)^[a-b]+c*(n|b)$/"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "node")
	assert.Equal(t, exprs[0].value, "/(?i)^[a-b]+c*(n|b)$/")

	// Allow space in value
	exprs, err = parseExprs([]string{"node==node 1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "node")
	assert.Equal(t, exprs[0].value, "node 1")
}

func TestMatch(t *testing.T) {
	e := expr{operator: EQ, value: "foo"}
	assert.True(t, e.Match("foo"))
	assert.False(t, e.Match("bar"))
	assert.True(t, e.Match("foo", "bar"))

	e = expr{operator: NOTEQ, value: "foo"}
	assert.False(t, e.Match("foo"))
	assert.True(t, e.Match("bar"))
	assert.False(t, e.Match("foo", "bar"))
	assert.False(t, e.Match("bar", "foo"))

	e = expr{operator: EQ, value: "f*o"}
	assert.True(t, e.Match("foo"))
	assert.True(t, e.Match("fuo"))
	assert.True(t, e.Match("foo", "fuo", "bar"))

	e = expr{operator: NOTEQ, value: "f*o"}
	assert.False(t, e.Match("foo"))
	assert.False(t, e.Match("fuo"))
	assert.False(t, e.Match("foo", "fuo", "bar"))
}
