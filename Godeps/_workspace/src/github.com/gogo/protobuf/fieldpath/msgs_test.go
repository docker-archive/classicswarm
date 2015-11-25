// Extensions for Protocol Buffers to create more go like structures.
//
// Copyright (c) 2013, Vastech SA (PTY) LTD. All rights reserved.
// http://github.com/gogo/protobuf/gogoproto
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package fieldpath_test

import (
	"github.com/gogo/protobuf/fieldpath"
	"github.com/gogo/protobuf/test"
	"reflect"
	"testing"
)

func TestToMessagesNinEmbeddedStruct(t *testing.T) {
	got, err := fieldpath.ToMessages("test.NinEmbeddedStruct.Field1", test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	exp := []string{"test.NinEmbeddedStruct", "test.NidOptNative"}
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Expected %v got %v", exp, got)
	}
}

func TestToMessagesMyExtendableFieldB(t *testing.T) {
	got, err := fieldpath.ToMessages("test.MyExtendable.FieldB", test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	exp := []string{"test.MyExtendable", "test.NinOptNative"}
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Expected %v got %v", exp, got)
	}
}

func TestToMessagesMyExtendableFieldCField1(t *testing.T) {
	got, err := fieldpath.ToMessages("test.MyExtendable.FieldC.Field1", test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	exp := []string{"test.MyExtendable", "test.NinEmbeddedStruct", "test.NidOptNative"}
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Expected %v got %v", exp, got)
	}
}
