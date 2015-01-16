package filter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidatingEnvExtraction(t *testing.T) {
	err := error(nil)

	// Cannot use the leading digit for key
	_, err = extractEnv("constraint", []string{"constraint:1node"})
	assert.Error(t, err)

	// Cannot use space in key
	_, err = extractEnv("constraint", []string{"constraint:node ==node1"})
	assert.Error(t, err)

	// Cannot use dot in key
	_, err = extractEnv("constraint", []string{"constraint:no.de==node1"})
	assert.Error(t, err)

	// Cannot use * in key
	_, err = extractEnv("constraint", []string{"constraint:no*de==node1"})
	assert.Error(t, err)

	// Allow leading underscore
	_, err = extractEnv("constraint", []string{"constraint:_node==_node1"})
	assert.NoError(t, err)

	// Allow globbing
	_, err = extractEnv("constraint", []string{"constraint:node==*node*"})
	assert.NoError(t, err)

	// Allow regexp in value
	_, err = extractEnv("constraint", []string{"constraint:node==/(?i)^[a-b]+c*$/"})
	assert.NoError(t, err)
}
