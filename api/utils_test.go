package api

import (
	"net/http"
	"net/url"
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
		r, err := http.NewRequest("POST", "", nil)
		assert.NoError(t, err)

		r.Form = v

		a := boolValue(r, "test")
		assert.Equal(t, a, e)
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
		r, err := http.NewRequest("POST", "", nil)
		assert.NoError(t, err)

		r.Form = v

		a := intValueOrZero(r, "test")
		assert.Equal(t, a, e)
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
		r, err := http.NewRequest("POST", "", nil)
		assert.NoError(t, err)

		r.Form = v

		a := int64ValueOrZero(r, "test")
		assert.Equal(t, a, e)
	}
}
