package api

import (
	"net/http"
	"net/url"
	"testing"
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

func TestMatchImageOSError(t *testing.T) {
	cases := map[string]string{
		`image operating system "linux" cannot be used on this platform`: "linux",
		`image operating system "" cannot be used on this platform`:      "",
		`not a matched string`: "",
	}

	for c, e := range cases {
		a := MatchImageOSError(c)
		if a != e {
			t.Fatalf("Value: %s, expected: %v, actual: %v", c, e, a)
		}
	}
}

func TestMatchHostConfigError(t *testing.T) {
	cases := map[string]string{
		// first, the real errors
		`Invalid isolation: "lxc" - linux only supports 'default'`:                        "linux",
		`Invalid QoS settings: linux does not support configuration of maximum bandwidth`: "linux",
		`Invalid QoS settings: linux does not support configuration of maximum IOPs`:      "linux",
		`Windows does not support CPU real-time period`:                                   "windows",
		`Windows does not support CPU real-time runtime`:                                  "windows",
		// then, some gibberish that should still match.
		`asl;dkfh some error Invalid isolation: "marchhare" - HAL only supports blasting dave into space`: "HAL",
		`Invalid QoS settings: someSystem does not support configuration battlebots`:                      "someSystem",
		`Windows does not support modding the kernel yeah that's right no open source`:                    "windows",
		// and a negative case, just to be sure.
		`Invalid not a real error. This shouldn't trigger anything`: "",
	}

	for test, expected := range cases {
		result := MatchHostConfigError(test)
		if result != expected {
			t.Fatalf("Value %s, expected %q, actual %q", test, expected, result)
		}
	}
}
