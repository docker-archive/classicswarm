package swarm

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertKVStringsToMap(t *testing.T) {
	result := convertKVStringsToMap([]string{"HELLO=WORLD", "a=b=c=d", "e"})
	expected := map[string]string{"HELLO": "WORLD", "a": "b=c=d", "e": ""}
	assert.Equal(t, expected, result)
}

func TestConvertMapToKVStrings(t *testing.T) {
	result := convertMapToKVStrings(map[string]string{"HELLO": "WORLD", "a": "b=c=d", "e": ""})
	sort.Strings(result)
	expected := []string{"HELLO=WORLD", "a=b=c=d", "e="}
	assert.Equal(t, expected, result)
}
