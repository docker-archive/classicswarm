package api

import (
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolValue(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"no":    false,
		"false": false,
		"none":  false,
		"1":     true,
		"yes":   true,
		"true":  true,
		"one":   true,
		"100":   true,
	}

	for c, e := range cases {
		v := url.Values{}
		v.Set("test", c)
		r, _ := http.NewRequest("POST", "", nil)
		r.Form = v

		a := boolValue(r, "test")
		if a != e {
			t.Fatalf("Value: %s, expected: %v, actual: %v", c, e, a)
		}
	}
}

func TestIntValueOrZero(t *testing.T) {
	cases := map[string]int{
		"":     0,
		"asdf": 0,
		"0":    0,
		"1":    1,
	}

	for c, e := range cases {
		v := url.Values{}
		v.Set("test", c)
		r, _ := http.NewRequest("POST", "", nil)
		r.Form = v

		a := intValueOrZero(r, "test")
		if a != e {
			t.Fatalf("Value: %s, expected: %v, actual: %v", c, e, a)
		}
	}
}

func TestInt64ValueOrZero(t *testing.T) {
	cases := map[string]int64{
		"":     0,
		"asdf": 0,
		"0":    0,
		"1":    1,
	}

	for c, e := range cases {
		v := url.Values{}
		v.Set("test", c)
		r, _ := http.NewRequest("POST", "", nil)
		r.Form = v

		a := int64ValueOrZero(r, "test")
		if a != e {
			t.Fatalf("Value: %s, expected: %v, actual: %v", c, e, a)
		}
	}
}

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
