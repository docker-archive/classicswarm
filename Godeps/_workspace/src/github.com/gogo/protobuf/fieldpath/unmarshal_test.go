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
	"fmt"
	"github.com/gogo/protobuf/fieldpath"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/test"
	math_rand "math/rand"
	"testing"
	"time"
)

func TestExtend(t *testing.T) {
	fp, err := fieldpath.NewFloat64Path("test", "MyExtendable", test.ThetestDescription(), "FieldA")
	if err != nil {
		panic(err)
	}
	m := &test.MyExtendable{}
	err = proto.SetExtension(m, test.E_FieldA, proto.Float64(10.0))
	if err != nil {
		panic(err)
	}
	buf, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	var unmarshalled float64
	f := FuncHandler{
		Float64Func: func(v float64) {
			t.Logf("unmarshalled %v", v)
			unmarshalled = v
		},
	}
	unmarshaler := fieldpath.NewFloat64Unmarshaler(fp, f)
	err = unmarshaler.Unmarshal(buf)
	if err != nil {
		panic(err)
	}
	if unmarshalled != float64(10.0) {
		panic(fmt.Errorf("wtf %v", unmarshalled))
	}
}

func BenchmarkUnmarshalNidOptNativeWhole(b *testing.B) {
	r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := test.NewPopulatedNidOptNative(r, false)
		data, err := proto.Marshal(p)
		if err != nil {
			panic(err)
		}
		b.StartTimer()
		pp := &test.NidOptNative{}
		proto.Unmarshal(data, pp)
	}
}

func BenchmarkUnmarshalNinOptStructWhole(b *testing.B) {
	r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := test.NewPopulatedNinOptStruct(r, false)
		data, err := proto.Marshal(p)
		if err != nil {
			panic(err)
		}
		b.StartTimer()
		pp := &test.NinOptStruct{}
		proto.Unmarshal(data, pp)
	}
}
