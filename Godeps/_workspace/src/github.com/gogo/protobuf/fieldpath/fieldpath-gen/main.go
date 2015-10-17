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

//Generates fieldpath-generated.go and fieldpath-generated_test.go in github.com/gogo/protobuf/fieldpath
//, since writing code for each native go type and native proto type can be quite tedious.
package main

import (
	"bytes"
	"go/format"
	"os"
	"text/template"
)

func out(t *template.Template, model interface{}) {
	buf := bytes.NewBuffer(nil)
	err := t.Execute(buf, model)
	if err != nil {
		panic(err)
	}
	bytes := buf.Bytes()
	expr, err := format.Source(bytes)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(expr)
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "test" {
			top := template.Must(template.New("top").Parse(NewTestTopTemplate()))
			out(top, nil)
			types := NewTypes()
			t := template.Must(template.New("types").Parse(NewTestTypeTemplate()))
			for _, typ := range types {
				out(t, typ)
			}
			return
		}
	}
	top := template.Must(template.New("top").Parse(NewTopTemplate()))
	out(top, nil)
	types := NewTypes()
	t := template.Must(template.New("types").Parse(NewTypeTemplate()))
	for _, typ := range types {
		if typ.packed {
			out(t, typ)
		}
		typ.packed = false
		out(t, typ)
	}
}
