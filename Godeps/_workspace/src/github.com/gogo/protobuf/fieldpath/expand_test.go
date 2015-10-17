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
	"testing"
)

func TestExpandAndCollapseOneLevel(t *testing.T) {
	collapsed := "Field1"
	expanded := "Field1.Field1"
	rootPkg := "test"
	rootMsg := "NinEmbeddedStruct"
	e, err := fieldpath.Expand(rootPkg, rootMsg, collapsed, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Expanded to %v", e)
	if e != expanded {
		t.Fatalf("Expected Expanded %v but got %v", expanded, e)
	}
	c, err := fieldpath.Collapse(rootPkg, rootMsg, expanded, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Collapsed to %v", c)
	if c != collapsed {
		t.Fatalf("Expected Collapsed %v but got %v", collapsed, c)
	}
}

func TestExpandAndCollapseExtendedField(t *testing.T) {
	collapsed := "FieldA"
	expanded := "FieldA"
	rootPkg := "test"
	rootMsg := "MyExtendable"
	e, err := fieldpath.Expand(rootPkg, rootMsg, collapsed, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Expanded to %v", e)
	if e != expanded {
		t.Fatalf("Expected Expanded %v but got %v", expanded, e)
	}
	c, err := fieldpath.Collapse(rootPkg, rootMsg, expanded, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Collapsed to %v", c)
	if c != collapsed {
		t.Fatalf("Expected Collapsed %v but got %v", collapsed, c)
	}
}

func TestExpandAndCollapseExtendedMessage(t *testing.T) {
	collapsed := "FieldB.Field1"
	expanded := "FieldB.Field1"
	rootPkg := "test"
	rootMsg := "MyExtendable"
	e, err := fieldpath.Expand(rootPkg, rootMsg, collapsed, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Expanded to %v", e)
	if e != expanded {
		t.Fatalf("Expected Expanded %v but got %v", expanded, e)
	}
	c, err := fieldpath.Collapse(rootPkg, rootMsg, expanded, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Collapsed to %v", c)
	if c != collapsed {
		t.Fatalf("Expected Collapsed %v but got %v", collapsed, c)
	}
}

func TestExpandAndCollapseExtendedMessageOneLevel(t *testing.T) {
	collapsed := "FieldC.Field1"
	expanded := "FieldC.Field1.Field1"
	rootPkg := "test"
	rootMsg := "MyExtendable"
	e, err := fieldpath.Expand(rootPkg, rootMsg, collapsed, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Expanded to %v", e)
	if e != expanded {
		t.Fatalf("Expected Expanded %v but got %v", expanded, e)
	}
	c, err := fieldpath.Collapse(rootPkg, rootMsg, expanded, test.ThetestDescription())
	if err != nil {
		panic(err)
	}
	t.Logf("Collapsed to %v", c)
	if c != collapsed {
		t.Fatalf("Expected Collapsed %v but got %v", collapsed, c)
	}
}
