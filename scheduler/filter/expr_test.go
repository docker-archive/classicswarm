package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExprs(t *testing.T) {
	// Cannot use the leading digit for key
	_, err := ParseExprs([]string{"1node"})
	assert.Error(t, err)

	// Cannot use space in key
	_, err = ParseExprs([]string{"node ==node1"})
	assert.Error(t, err)

	// Cannot use * in key
	_, err = ParseExprs([]string{"no*de==node1"})
	assert.Error(t, err)

	// Cannot use $ in key
	_, err = ParseExprs([]string{"no$de==node1"})
	assert.Error(t, err)

	// Allow CAPS in key
	exprs, err := ParseExprs([]string{"NoDe==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "NoDe")
	assert.Equal(t, exprs[0].Value, "node1")

	// Allow dot in key
	exprs, err = ParseExprs([]string{"no.de==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "no.de")
	assert.Equal(t, exprs[0].Value, "node1")

	// Allow leading underscore
	exprs, err = ParseExprs([]string{"_node==_node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "_node")
	assert.Equal(t, exprs[0].Value, "_node1")

	// Allow globbing
	exprs, err = ParseExprs([]string{"node==*node*"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "node")
	assert.Equal(t, exprs[0].Value, "*node*")

	// Allow regexp in value
	exprs, err = ParseExprs([]string{"node==/(?i)^[a-b]+c*(n|b)$/"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "node")
	assert.Equal(t, exprs[0].Value, "/(?i)^[a-b]+c*(n|b)$/")

	// Allow space in value
	exprs, err = ParseExprs([]string{"node==node 1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].Key, "node")
	assert.Equal(t, exprs[0].Value, "node 1")
}

func TestMatch(t *testing.T) {
	e := Expr{Operator: EQ, Value: "foo"}
	assert.True(t, e.Match("foo"))
	assert.False(t, e.Match("bar"))
	assert.True(t, e.Match("foo", "bar"))

	e = Expr{Operator: NOTEQ, Value: "foo"}
	assert.False(t, e.Match("foo"))
	assert.True(t, e.Match("bar"))
	assert.False(t, e.Match("foo", "bar"))
	assert.False(t, e.Match("bar", "foo"))

	e = Expr{Operator: EQ, Value: "f*o"}
	assert.True(t, e.Match("foo"))
	assert.True(t, e.Match("fuo"))
	assert.True(t, e.Match("foo", "fuo", "bar"))

	e = Expr{Operator: NOTEQ, Value: "f*o"}
	assert.False(t, e.Match("foo"))
	assert.False(t, e.Match("fuo"))
	assert.False(t, e.Match("foo", "fuo", "bar"))
}
