package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExprs(t *testing.T) {
	// Cannot use the leading digit for key
	_, err := parseExprs("constraint", []string{"constraint:1node"})
	assert.Error(t, err)

	// Cannot use space in key
	_, err = parseExprs("constraint", []string{"constraint:node ==node1"})
	assert.Error(t, err)

	// Cannot use * in key
	_, err = parseExprs("constraint", []string{"constraint:no*de==node1"})
	assert.Error(t, err)

	// Cannot use $ in key
	_, err = parseExprs("constraint", []string{"constraint:no$de==node1"})
	assert.Error(t, err)

	// Allow CAPS in key
	_, err = parseExprs("constraint", []string{"constraint:NoDe==node1"})
	assert.NoError(t, err)

	// Allow dot in key
	_, err = parseExprs("constraint", []string{"constraint:no.de==node1"})
	assert.NoError(t, err)

	// Allow leading underscore
	_, err = parseExprs("constraint", []string{"constraint:_node==_node1"})
	assert.NoError(t, err)

	// Allow globbing
	_, err = parseExprs("constraint", []string{"constraint:node==*node*"})
	assert.NoError(t, err)

	// Allow regexp in value
	_, err = parseExprs("constraint", []string{"constraint:node==/(?i)^[a-b]+c*(n|b)$/"})
	assert.NoError(t, err)

	// Allow space in value
	_, err = parseExprs("constraint", []string{"constraint:node==node 1"})
	assert.NoError(t, err)
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
