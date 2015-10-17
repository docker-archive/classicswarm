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
	"encoding/binary"
	"github.com/gogo/protobuf/fieldpath"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/test"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestNoMergeNoMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := test.NewPopulatedNinOptNative(r, true)
	data, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "NinOptNative")
	if err != nil {
		panic(err)
	}
}

func TestNoMergeMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := test.NewPopulatedNinOptNative(r, true)
	if m.Field1 == nil {
		m.Field1 = proto.Float64(1.1)
	}
	data, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	key := byte(uint32(1)<<3 | uint32(1))
	data = append(data, key, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "NinOptNative")
	if err == nil || !strings.Contains(err.Error(), "NinOptNative.Field1 requires merging") {
		t.Fatalf("Field1 should require merging")
	}
}

func TestNoMergeNestedNoMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bigm := test.NewPopulatedNidOptStruct(r, true)
	data, err := proto.Marshal(bigm)
	if err != nil {
		panic(err)
	}
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "NidOptStruct")
	if err != nil {
		panic(err)
	}
}

func TestNoMergeMessageMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bigm := test.NewPopulatedNidOptStruct(r, true)
	data, err := proto.Marshal(bigm)
	if err != nil {
		panic(err)
	}
	key := byte(uint32(4)<<3 | uint32(2))
	data = append(data, key, 5, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "NidOptStruct")
	if err == nil || !strings.Contains(err.Error(), "requires merging") {
		panic(err)
	}
}

func TestNoMergeNestedMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bigm := test.NewPopulatedNidOptStruct(r, true)
	m := test.NewPopulatedNinOptNative(r, true)
	if m.Field1 == nil {
		m.Field1 = proto.Float64(1.1)
	}
	key := byte(uint32(1)<<3 | uint32(1))
	m.XXX_unrecognized = append(m.XXX_unrecognized, key, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	bigm.Field4 = *m
	data, err := proto.Marshal(bigm)
	if err != nil {
		panic(err)
	}
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "NidOptStruct")
	if err == nil || !strings.Contains(err.Error(), "NinOptNative.Field1 requires merging") {
		t.Fatalf("Field1 should require merging")
	}
}

func TestNoMergeExtensionNoMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bigm := test.NewPopulatedMyExtendable(r, true)
	m := test.NewPopulatedNinOptNative(r, true)
	err := proto.SetExtension(bigm, test.E_FieldB, m)
	if err != nil {
		panic(err)
	}
	data, err := proto.Marshal(bigm)
	if err != nil {
		panic(err)
	}
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "MyExtendable")
	if err != nil {
		panic(err)
	}
}

func TestNoMergeExtensionMerge(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bigm := test.NewPopulatedMyExtendable(r, true)
	m := test.NewPopulatedNinOptNative(r, true)
	err := proto.SetExtension(bigm, test.E_FieldB, m)
	if err != nil {
		panic(err)
	}
	data, err := proto.Marshal(bigm)
	if err != nil {
		panic(err)
	}
	key := uint32(101)<<3 | uint32(2)
	data2 := make([]byte, 10)
	n := binary.PutUvarint(data2, uint64(key))
	data2 = data2[:n]
	data = append(data, data2...)
	data4, err := proto.Marshal(test.NewPopulatedNinOptNative(r, true))
	if err != nil {
		panic(err)
	}
	data3 := make([]byte, 10)
	n = binary.PutUvarint(data3, uint64(len(data4)))
	data3 = data3[:n]
	data = append(data, data3...)
	data = append(data, data4...)
	err = fieldpath.NoMerge(data, test.ThetestDescription(), "test", "MyExtendable")
	if err == nil || !strings.Contains(err.Error(), "requires merging") {
		t.Fatalf("should require merging")
	}
}
