// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"bytes"
	"testing"
)

func TestThatPBKDFWorksCorrectlyForLongKeys(t *testing.T) {
	cipherInfo := shaWithTripleDESCBC{}

	salt := []byte("\xff\xff\xff\xff\xff\xff\xff\xff")
	password, _ := bmpString("sesame")
	key := cipherInfo.deriveKey(salt, password, 2048)

	if expected := []byte("\x7c\xd9\xfd\x3e\x2b\x3b\xe7\x69\x1a\x44\xe3\xbe\xf0\xf9\xea\x0f\xb9\xb8\x97\xd4\xe3\x25\xd9\xd1"); bytes.Compare(key, expected) != 0 {
		t.Fatalf("expected key '%x', but found '%x'", key, expected)
	}
}
