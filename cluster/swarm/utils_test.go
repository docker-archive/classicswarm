package swarm

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func removePointersFromMap(values map[string]*string) map[string]string {
	result := make(map[string]string, len(values))

	for key, val := range values {
		valueString := "NILPOINTER"
		if val != nil {
			valueString = *val
		}
		result[key] = valueString
	}
	return result
}

func TestConvertKVStringsToMap(t *testing.T) {
	result := convertKVStringsToMap([]string{"HELLO=WORLD", "a=b=c=d", "e"})
	expected := map[string]string{"HELLO": "WORLD", "a": "b=c=d", "e": "NILPOINTER"}
	assert.Equal(t, expected, removePointersFromMap(result))
}

func TestConvertMapToKVStrings(t *testing.T) {
	helloString := "WORLD"
	aString := "b=c=d"
	eString := ""
	result := convertMapToKVStrings(map[string]*string{"HELLO": &helloString, "a": &aString, "e": &eString, "f": nil})
	sort.Strings(result)
	expected := []string{"HELLO=WORLD", "a=b=c=d", "e=", "f="}
	assert.Equal(t, expected, result)
}
