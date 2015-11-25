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

package main

import (
	"strings"
)

type TestType struct {
	Struct   string
	Path     string
	Repeated bool
	packed   bool
	Single   bool
	Enum     bool
}

func (this *TestType) NotNil(root string) string {
	paths := strings.Split(this.Path, ".")
	notnil := make([]string, len(paths))
	for i := range paths {
		notnil[i] = root + "." + strings.Join(paths[:i+1], ".") + " != nil"
	}
	return strings.Join(notnil, " && ")
}

func (this *TestType) CPacked() string {
	if this.packed {
		return "Packed"
	}
	return ""
}

func (this *TestType) Packed() string {
	if this.packed {
		return "packed"
	}
	return ""
}

type Type struct {
	Name      string
	GoTyp     string
	CodeParts []string
	Return    string
	N         string
	packed    bool
	Slice     bool
	Tests     []TestType
	ProtoTyp  []string
	Sort      bool
}

func NewTypes() []*Type {
	return []*Type{
		{
			Name:      "float64",
			ProtoTyp:  []string{"DOUBLE"},
			GoTyp:     "float64",
			CodeParts: []string{`if endOf < offset+8 {`, `}`},
			Return:    `*(*float64)(unsafe.Pointer(&buf[offset]))`,
			N:         `8`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field1",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field1",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field1",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field1",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field1",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field1",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:      "float32",
			ProtoTyp:  []string{"FLOAT"},
			GoTyp:     "float32",
			CodeParts: []string{`if endOf < offset+4 {`, `}`},
			Return:    `*(*float32)(unsafe.Pointer(&buf[offset]))`,
			N:         `4`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field2",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field2",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field2",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field2",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field2",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field2",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "int32",
			ProtoTyp: []string{"INT32", "ENUM"},
			GoTyp:    "int32",
			CodeParts: []string{`var v int32
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}`},
			Return: `v`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field3",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field3",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field3",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field3",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field3",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field3",
					Single: true,
				},
				{
					Struct: "NinOptEnumDefault",
					Path:   "Field1",
					Single: true,
					Enum:   true,
				},
				{
					Struct: "AnotherNinOptEnum",
					Path:   "Field1",
					Single: true,
					Enum:   true,
				},
			},
			Sort: true,
		},
		{
			Name:     "int64",
			ProtoTyp: []string{"INT64"},
			GoTyp:    "int64",
			CodeParts: []string{`var v int64
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}`},
			Return: `v`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field4",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field4",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field4",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field4",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field4",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field4",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "uint64",
			ProtoTyp: []string{"UINT64"},
			GoTyp:    "uint64",
			CodeParts: []string{`var v uint64
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}`},
			Return: `v`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field6",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field6",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field6",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field6",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field6",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field6",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "uint32",
			ProtoTyp: []string{"UINT32"},
			GoTyp:    "uint32",
			CodeParts: []string{`var v uint32
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}`},
			Return: `v`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field5",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field5",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field5",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field5",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field5",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field5",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "sint32",
			ProtoTyp: []string{"SINT32"},
			GoTyp:    "int32",
			CodeParts: []string{`var v int32
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			vv := int32((uint32(v) >> 1) ^ uint32(((v&1)<<31)>>31))`},
			Return: `vv`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field7",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field7",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field7",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field7",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field7",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field7",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "sint64",
			ProtoTyp: []string{"SINT64"},
			GoTyp:    "int64",
			CodeParts: []string{`var v uint64
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`, `
				}
				b := buf[offset+n]
				n++
				v |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			vv := int64((v >> 1) ^ uint64((int64(v&1)<<63)>>63))`},
			Return: `vv`,
			N:      `n`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field8",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field8",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field8",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field8",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field8",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field8",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:      "fixed32",
			ProtoTyp:  []string{"FIXED32"},
			GoTyp:     "uint32",
			CodeParts: []string{`if endOf < offset+4 {`, `}`},
			Return:    `*(*uint32)(unsafe.Pointer(&buf[offset]))`,
			N:         `4`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field9",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field9",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field9",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field9",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field9",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field9",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:      "fixed64",
			ProtoTyp:  []string{"FIXED64"},
			GoTyp:     "uint64",
			CodeParts: []string{`if endOf < offset+8 {`, `}`},
			Return:    `*(*uint64)(unsafe.Pointer(&buf[offset]))`,
			N:         `8`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field11",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field11",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field11",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field11",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field11",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field11",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:      "sfixed32",
			ProtoTyp:  []string{"SFIXED32"},
			GoTyp:     "int32",
			CodeParts: []string{`if endOf < offset+4 {`, `}`},
			Return:    `*(*int32)(unsafe.Pointer(&buf[offset]))`,
			N:         `4`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field10",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field10",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field10",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field10",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field10",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field10",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:      "sfixed64",
			ProtoTyp:  []string{"SFIXED64"},
			GoTyp:     "int64",
			CodeParts: []string{`if endOf < offset+8 {`, `}`},
			Return:    `*(*int64)(unsafe.Pointer(&buf[offset]))`,
			N:         `8`,
			packed:    true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field12",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field12",
					Single: true,
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field12",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field12",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field12",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field12",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "bool",
			ProtoTyp: []string{"BOOL"},
			GoTyp:    "bool",
			CodeParts: []string{`var v int32
			n := 0
			for shift := uint(0); ; shift += 7 {
				if offset+n >= endOf {
				`,
				`
				}
				b := buf[offset+n]
				n++
				v |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			vv := v != 0`},
			Return: `vv`,
			N:      "n",
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field13",
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field13",
				},
				{
					Struct:   "NinRepPackedNative",
					Path:     "Field13",
					Repeated: true,
					packed:   true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field13",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field13",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field13",
				},
			},
			Sort: false,
		},
		{
			Name:     "string",
			ProtoTyp: []string{"STRING"},
			GoTyp:    "string",
			CodeParts: []string{
				`var stringLen uint64
				n := 0
				for shift := uint(0); ; shift += 7 {
					if offset >= endOf {`,
				`}
					b := buf[offset+n]
					n++
					stringLen |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				postIndex := offset + n + int(stringLen)
				if postIndex > endOf {`,
				`}
				vv := string(buf[offset+n:postIndex])`},
			Return: `vv`,
			N:      `n + int(stringLen)`,
			packed: true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field14",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field14",
					Single: true,
				},
				{
					Struct:   "NinRepNative",
					Path:     "Field14",
					Repeated: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field14",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field14",
					Single: true,
				},
			},
			Sort: true,
		},
		{
			Name:     "bytes",
			ProtoTyp: []string{"BYTES", "MESSAGE"},
			GoTyp:    "[]byte",
			CodeParts: []string{
				`var bytesLen uint64
				n := 0
				for shift := uint(0); ; shift += 7 {
					if offset >= endOf {`,
				`}
					b := buf[offset+n]
					n++
					bytesLen |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				postIndex := offset + n + int(bytesLen)
				if postIndex > endOf {`,
				`}
				vv := buf[offset+n:postIndex]`},
			Return: `vv`,
			N:      `n + int(bytesLen)`,
			packed: true,
			Slice:  true,
			Tests: []TestType{
				{
					Struct: "NinOptNative",
					Path:   "Field15",
					Single: true,
				},
				{
					Struct: "NinOptStruct",
					Path:   "Field4.Field15",
					Single: true,
				},
				{
					Struct: "NinNestedStruct",
					Path:   "Field1.Field4.Field15",
				},
				{
					Struct: "NinOptNativeDefault",
					Path:   "Field15",
					Single: true,
				},
			},
			Sort: true,
		},
	}
}

func (this *Type) CName() string {
	return strings.ToUpper(this.Name[0:1]) + this.Name[1:]
}

func (this *Type) CGoTyp() string {
	if this.Slice {
		return strings.ToUpper(this.GoTyp[2:3]) + this.GoTyp[3:] + "s"
	}
	return strings.ToUpper(this.GoTyp[0:1]) + this.GoTyp[1:]
}

func (this *Type) ReturnValue() string {
	if this.Slice {
		return this.Return
	}
	if strings.HasPrefix(this.Return, "*") {
		return this.Return[1:]
	}
	return "&" + this.Return
}

func (this *Type) ReturnType() string {
	if this.Slice {
		return this.GoTyp
	}
	return "*" + this.GoTyp
}

func (this *Type) DecCode() string {
	return strings.Join(this.CodeParts, "return 0, io.ErrUnexpectedEOF")
}

func (this *Type) UnmarshalCode() string {
	return strings.Join(this.CodeParts, "return nil, io.ErrUnexpectedEOF")
}

func (this *Type) Bytes() bool {
	return this.GoTyp == "[]byte"
}

func (this *Type) Packed() string {
	if this.packed {
		return "packed"
	}
	return ""
}

func (this *Type) CPacked() string {
	if this.packed {
		return "Packed"
	}
	return ""
}

func NewTopTemplate() string {
	return `// Code generated by fieldpath-gen.
	// source: github.com/gogo/protobuf/fieldpath/fieldpath-gen/template.go
	// DO NOT EDIT!

	package fieldpath
	import "io"
	import "unsafe"
	import "bytes"
	import descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	var _ = bytes.MinRead
	`
}

func NewTypeTemplate() string {
	return `
{{if .Packed}}
//Decodes a repeated packed field and sends the elements one by one to the handler.
//The buffer, from the offset, should start with the varint encoded length followed by the list of encoded values.
//The number of bytes consumed after the offset is returned from this function.
func DecPacked{{.CName}}(buf []byte, offset int, handler {{.CGoTyp}}Handler) (int, error) {
	length, n, err := decodeVarint(buf, offset)
	if err != nil {
		return 0, err
	}
	offset = offset + n
	nn := n + int(length)
	endOf := offset + int(length)
	for offset < endOf {
		n, err = Dec{{.CName}}(buf, offset, handler)
		if err != nil {
			return 0, err
		}
		offset += n
	}
	return nn, nil
}
{{else}}
//Decodes a protocol buffer encoded value and sends the value to the handler.
//The number of bytes consumed after the offset is returned from this function.
func Dec{{.CName}}(buf []byte, offset int, handler {{.CGoTyp}}Handler) (int, error) {
	endOf := len(buf)
	{{.DecCode}}
	handler.{{.CGoTyp}}({{.Return}})
	return {{.N}}, nil
}
{{end}}

//Contains the ordered list of keys, compiled path.
type {{.CPacked}}{{.CName}}Path struct {
	path []uint64
	def  {{.ReturnType}}
}

//Returns the ordered list of keys, compiled path.
func (this *{{.CPacked}}{{.CName}}Path) GetPath() []uint64 {
	return this.path
}

//Returns this default value of the field
func (this *{{.CPacked}}{{.CName}}Path) GetDefault() {{.ReturnType}} {
	return this.def
}

//This constructor also checks if the path is valid and the type in the descriptor is matches the called function.
//This function should preferably be called in the init of a module, since this is not very type safe (stringly typed), this will help in catching these errors sooner rather than later.
func New{{.CPacked}}{{.CName}}Path(rootPackage string, rootMessage string, descSet *descriptor.FileDescriptorSet, path string) (*{{.CPacked}}{{.CName}}Path, error) {
	fd, err := new(rootPackage, rootMessage, descSet, path)
	if err != nil {
		return nil, err
	}
	{{$this := .}}
	{{range .ProtoTyp}}
	if fd.field.GetType() == descriptor.FieldDescriptorProto_TYPE_{{.}} {
		{{if $this.Packed}}
		return &{{$this.CPacked}}{{$this.CName}}Path{fd.path, nil}, nil
		{{else}}
		{{if $this.Bytes}}
		return &{{$this.CPacked}}{{$this.CName}}Path{fd.path, nil}, nil
		{{else}}
		return &{{$this.CPacked}}{{$this.CName}}Path{fd.path, fd.GetDefault{{$this.CGoTyp}}()}, nil
		{{end}}
		{{end}}
	}
	{{end}}
	return nil, &errType{descriptor.FieldDescriptorProto_TYPE_{{index .ProtoTyp 0}}, fd.field.GetType()}
}

{{if .Packed}}
type {{.Packed}}{{.CName}}Unmarshaler struct {
	handler {{.CGoTyp}}Handler
}

func (this *{{.Packed}}{{.CName}}Unmarshaler) unmarshal(buf []byte, offset int) (int, error) {
	length, n, err := decodeVarint(buf, offset)
	if err != nil {
		return 0, err
	}
	offset = offset + n
	nn := n + int(length)
	endOf := offset + int(length)
	for offset < endOf {
		n, err = Dec{{.CName}}(buf, offset, this.handler)
		if err != nil {
			return 0, err
		}
		offset += n
	}
	return nn, nil
}

func (this *{{.Packed}}{{.CName}}Unmarshaler) reset() {}

func (this *{{.Packed}}{{.CName}}Unmarshaler) unmarshalDefault() {}

{{else}}
type {{.Name}}Unmarshaler struct {
	handler {{.CGoTyp}}Handler
	def {{.ReturnType}}
	set bool
}

func (this *{{.Name}}Unmarshaler) unmarshal(buf []byte, offset int) (int, error) {
	this.set = true
	endOf := len(buf)
	{{.DecCode}}
	this.handler.{{.CGoTyp}}({{.Return}})
	return {{.N}}, nil
}

func (this *{{.Name}}Unmarshaler) reset() {
	this.set = false
}

func (this *{{.Name}}Unmarshaler) unmarshalDefault() {
	if this.def != nil && !this.set {
		{{if .Bytes}}
		this.handler.{{.CGoTyp}}(this.def)	
		{{else}}
		this.handler.{{.CGoTyp}}(*this.def)	
		{{end}}
	}
}
{{end}}


func new{{.CPacked}}{{.CName}}Unmarshaler(path []uint64, def {{.ReturnType}}, h {{.CGoTyp}}Handler) *Unmarshaler {
	return &Unmarshaler{
		{{if .Packed}}
		unmarshaler: &{{.Packed}}{{.CName}}Unmarshaler{h},
		{{else}}
		unmarshaler: &{{.Name}}Unmarshaler{h, def, false},
		{{end}}
		path:    path,
	}
}

//Combines a {{.CPacked}}{{.CName}}Path and a {{.CGoTyp}}Handler into an Unmarshaler
func New{{.CPacked}}{{.CName}}Unmarshaler(f *{{.CPacked}}{{.CName}}Path, h {{.CGoTyp}}Handler) *Unmarshaler {
	return new{{.CPacked}}{{.CName}}Unmarshaler(f.GetPath(), f.GetDefault(), h)
}

{{if .Packed}}
{{else}}
{{if .Sort}}

//Contains an ordered key list, compiled path, for a single value.
type {{.CName}}SinglePath struct {
	path []uint64
	def  {{.ReturnType}}
}

//Returns an ordered key list, previously compiled path, for a single value.
func (this *{{.CName}}SinglePath) GetPath() []uint64 {
	return this.path
}

//Returns this default value of the field
func (this *{{.CName}}SinglePath) GetDefault() {{.ReturnType}} {
	return this.def
}

//This constructor also checks if the path is valid and the type in the descriptor is matches the called function.
//This function should preferably be called in the init of a module, since this is not very type safe (stringly typed), this will help in catching these errors sooner rather than later.
//This function also checks that there are no repeated fields on the path and that there should only be one value on this path.
//One value, really means zero or more values, where zero occurences means nil, one occurence is trivial and more than one occurence means that the last occurence overwrites the previous occurences.
func New{{.CName}}SinglePath(rootPackage string, rootMessage string, descSet *descriptor.FileDescriptorSet, path string) (*{{.CName}}SinglePath, error) {
	fd, err := new(rootPackage, rootMessage, descSet, path)
	if err != nil {
		return nil, err
	}
	for _, f := range fd.fields {
		if f.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			return nil, &errRepeated{path: path, fieldName: f.GetName()}
		}
	}
	{{$this := .}}
	{{range .ProtoTyp}}
	if fd.field.GetType() == descriptor.FieldDescriptorProto_TYPE_{{.}} {
		{{if $this.Bytes}}
		return &{{$this.CName}}SinglePath{fd.path, nil}, nil
		{{else}}
		return &{{$this.CName}}SinglePath{fd.path, fd.GetDefault{{$this.CGoTyp}}()}, nil
		{{end}}
	}
	{{end}}
	return nil, &errType{descriptor.FieldDescriptorProto_TYPE_{{index .ProtoTyp 0}}, fd.field.GetType()}
}

//Technically this will work for most but not all protocol buffers, depends on how it was marshalled.
//Use with caution.
func (this *{{.CName}}SinglePath) UnmarshalFirst(buf []byte) ({{.ReturnType}}, error) {
	position := 0
	final := len(this.path) - 1
	offset := 0
	endOfs := []int{len(buf)}
	endOf := endOfs[position]
	for position > 0 || offset < endOfs[0] {
		key, n, err := decodeVarint(buf, offset)
		if err != nil {
			return nil, err
		}
		wireType := int(key & 0x7)
		pp := this.path[position]
		offset+=n
		if pp == key {
			if final == position {
				{{.UnmarshalCode}}
				return {{.ReturnValue}}, nil
			}
			position++
			length, n, err := decodeVarint(buf, offset)
			if err != nil {
				return nil, err
			}
			offset = offset + n
			if len(endOfs) == position {
				endOfs = append(endOfs, offset+int(length))
			} else {
				endOfs[position] = offset + int(length)
			}
		} else {
			offset, err = skip(buf, offset, wireType)
			if err != nil {
				return nil, err
			}
		}
		for position > 0 && offset == endOfs[position] {
			position--
			endOf = endOfs[position]
		}
	}
	return this.def, nil
}

//Technically UnmarshalLast which is the correct protocol buffer compliant way to unmarshal only one field.
func (this *{{.CName}}SinglePath) Unmarshal(buf []byte) ({{.ReturnType}}, error) {
	var ret {{.ReturnType}} = this.def
	position := 0
	final := len(this.path) - 1
	offset := 0
	endOfs := []int{len(buf)}
	endOf := endOfs[position]
	for position > 0 || offset < endOfs[0] {
		key, n, err := decodeVarint(buf, offset)
		if err != nil {
			return nil, err
		}
		wireType := int(key & 0x7)
		pp := this.path[position]
		offset+=n
		if pp == key {
			if final == position {
				{{.UnmarshalCode}}
				ret = {{.ReturnValue}}
				offset = offset + {{.N}}
			} else {
				position++
				length, n, err := decodeVarint(buf, offset)
				if err != nil {
					return nil, err
				}
				offset = offset + n
				if len(endOfs) == position {
					endOfs = append(endOfs, offset+int(length))
				} else {
					endOfs[position] = offset + int(length)
				}
			}
		} else {
			offset, err = skip(buf, offset, wireType)
			if err != nil {
				return nil, err
			}
		}
		for position > 0 && offset == endOfs[position] {
			position--
			endOf = endOfs[position]
		}
	}
	return ret, nil
}

//Used to sort marshalled protocol buffers on a single {{.CName}} field.
//Provides memoizing to avoid unmarshaling the same value more than once.
type {{.CName}}Sorter struct {
	Sort
	path  *{{.CName}}SinglePath
	mem   []{{.ReturnType}}
	saved []bool
}

func New{{.CName}}Sorter(list Sort, path *{{.CName}}SinglePath) *{{.CName}}Sorter {
	return &{{.CName}}Sorter{
		Sort:  list,
		path:  path,
		mem:   make([]{{.ReturnType}}, list.Len()),
		saved: make([]bool, list.Len()),
	}
}

func (this *{{.CName}}Sorter) Less(i, j int) bool {
	var err error
	if !this.saved[i] {
		this.mem[i], err = this.path.Unmarshal(this.Get(i))
		if err != nil {
			panic(err)
		}
		this.saved[i] = true
	}
	if !this.saved[j] {
		this.mem[j], err = this.path.Unmarshal(this.Get(j))
		if err != nil {
			panic(err)
		}
		this.saved[j] = true
	}
	if this.mem[j] == nil {
		return false
	}
	if this.mem[i] == nil {
		return true
	}
	{{if .Bytes}}
	return bytes.Compare(this.mem[i], this.mem[j]) == -1
	{{else}}
	return *this.mem[i] < *this.mem[j]
	{{end}}
}

func (this *{{.CName}}Sorter) Swap(i, j int) {
	this.Sort.Swap(i, j)
	this.mem[i], this.mem[j] = this.mem[j], this.mem[i]
	this.saved[i], this.saved[j] = this.saved[j], this.saved[i]
}
{{end}}

{{end}}
`
}

func NewTestTopTemplate() string {
	return `// Code generated by fieldpath-gen.
	// source: github.com/gogo/protobuf/fieldpath/fieldpath-gen/template.go
	// DO NOT EDIT!
	
	package fieldpath_test
	import (
		"github.com/gogo/protobuf/fieldpath"
		"github.com/gogo/protobuf/proto"
		"github.com/gogo/protobuf/test"
		math_rand "math/rand"
		"testing"
		"time"
		"sort"
		"bytes"
		"fmt"
		"reflect"
	)
	var _ = bytes.MinRead
	var pseudo int64 = time.Now().UnixNano()

type FuncHandler struct {
	Float64Func
	Float32Func
	Int64Func
	Int32Func
	Uint64Func
	Uint32Func
	BoolFunc
	BytesFunc
	StringFunc
}

type Float64Func func(float64)
type Float32Func func(float32)
type Int64Func func(int64)
type Int32Func func(int32)
type Uint64Func func(uint64)
type Uint32Func func(uint32)
type BoolFunc func(bool)
type BytesFunc func([]byte)
type StringFunc func(string)

func (this Float64Func) Float64(v float64) { this(v) }
func (this Float32Func) Float32(v float32) { this(v) }
func (this Int64Func) Int64(v int64)       { this(v) }
func (this Int32Func) Int32(v int32)       { this(v) }
func (this Uint64Func) Uint64(v uint64)    { this(v) }
func (this Uint32Func) Uint32(v uint32)    { this(v) }
func (this BoolFunc) Bool(v bool)          { this(v) }
func (this BytesFunc) Bytes(v []byte)      { this(v) }
func (this StringFunc) String(v string)    { this(v) }

type sortable struct {
	list [][]byte
}

func (this *sortable) Len() int {
	return len(this.list)
}

func (this *sortable) Swap(i, j int) {
	this.list[i], this.list[j] = this.list[j], this.list[i]
}

func (this *sortable) Get(index int) []byte {
	return this.list[index]
}

	`
}

func NewTestTypeTemplate() string {
	return `{{$this := .}}
	{{range .Tests}}

	{{if .Single}}
		func TestUnmarshalFirst{{.Struct}}{{$this.CName}}(t *testing.T) {
			if testing.Short() {
				t.Skip("skipping test in short mode.")
			}
			r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
			fp, err := fieldpath.New{{$this.CName}}SinglePath("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
			if err != nil {
				panic(err)
			}
			p := test.NewPopulated{{.Struct}}(r, false)
			buf, err := proto.Marshal(p)
			if err != nil {
				panic(err)
			}
			unmarshalled, err := fp.UnmarshalFirst(buf)
			if err != nil {
				panic(err)
			}
			if !({{.NotNil "p"}}) {
				if unmarshalled != nil {
					{{if $this.Bytes}}
					{{else}}
					if *unmarshalled == *fieldpath.TestDefault("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}").GetDefault{{$this.CGoTyp}}() {
						return
					}
					{{end}}
					t.Fatalf("unmarshalled != nil")
				}
				return
			}
			if unmarshalled == nil {
				t.Fatalf("ummarshalled == nil")
			}
			{{if $this.Bytes}}if !bytes.Equal(unmarshalled, p.{{.Path}}) {
				t.Fatalf("%v != %v", unmarshalled, p.{{.Path}})
			}
			{{else}}
			{{if .Enum}}if int32(*unmarshalled) != int32(*p.{{.Path}}) {
				t.Fatalf("%v != %v", *unmarshalled, *p.{{.Path}})
			}{{else}}if *unmarshalled != *p.{{.Path}} {
				t.Fatalf("%v != %v", *unmarshalled, *p.{{.Path}})
			}{{end}}
			{{end}}
		}

		func TestUnmarshal{{.Struct}}{{$this.CName}}(t *testing.T) {
			if testing.Short() {
				t.Skip("skipping test in short mode.")
			}
			r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))	
			fp, err := fieldpath.New{{$this.CName}}SinglePath("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
			if err != nil {
				panic(err)
			}
			p := test.NewPopulated{{.Struct}}(r, false)
			buf, err := proto.Marshal(p)
			if err != nil {
				panic(err)
			}
			unmarshalled, err := fp.Unmarshal(buf)
			if err != nil {
				panic(err)
			}
			if !({{.NotNil "p"}}) {
				if unmarshalled != nil {
					{{if $this.Bytes}}
					{{else}}
					if *unmarshalled == *fieldpath.TestDefault("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}").GetDefault{{$this.CGoTyp}}() {
						return
					}
					{{end}}
					t.Fatalf("unmarshalled != nil")
				}
				return
			}
			if unmarshalled == nil {
				t.Fatalf("ummarshalled == nil")
			}
			{{if $this.Bytes}}if !bytes.Equal(unmarshalled, p.{{.Path}}) {
				t.Fatalf("%v != %v", unmarshalled, p.{{.Path}})
			}
			{{else}}
			{{if .Enum}}if int32(*unmarshalled) != int32(*p.{{.Path}}) {
				t.Fatalf("%v != %v", *unmarshalled, *p.{{.Path}})
			}{{else}}if *unmarshalled != *p.{{.Path}} {
				t.Fatalf("%v != %v", *unmarshalled, *p.{{.Path}})
			}{{end}}
			{{end}}
		}
	{{end}}

	func TestCompiled{{.Struct}}{{$this.CName}}(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}
		r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
		fp, err := fieldpath.New{{.CPacked}}{{$this.CName}}Path("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
		if err != nil {
			panic(err)
		}
		p := test.NewPopulated{{.Struct}}(r, false)
		var unmarshalled []{{$this.GoTyp}}
		f := FuncHandler{
			{{$this.CGoTyp}}Func: func(v {{$this.GoTyp}}) {
				unmarshalled = append(unmarshalled, v)
			},
		}
		unmarshaler := fieldpath.New{{.CPacked}}{{$this.CName}}Unmarshaler(fp, f)
		buf, err := proto.Marshal(p)
		if err != nil {
			panic(err)
		}
		compiled := fieldpath.Compile(unmarshaler)
		err = compiled.Unmarshal(buf)
		if err != nil {
			panic(err)
		}
		{{if .Repeated}}
			{{if $this.Bytes}}panic("not implemented")
			{{else}}if !reflect.DeepEqual(unmarshalled, p.{{.Path}}) {
				if len(unmarshalled) == 0 && len(p.{{.Path}}) == 0 {
					return
				}
				panic(fmt.Errorf("unmarshalled %#v != p.{{.Path}} %#v", unmarshalled, p.{{.Path}}))
			}
			{{end}}
		{{else}}
			if {{.NotNil "p"}} {
				{{if .Enum}}
				if !reflect.DeepEqual(int32(unmarshalled[0]), int32({{if $this.Bytes}}{{else}}*{{end}}p.{{.Path}})) {
				{{else}}
				if !reflect.DeepEqual(unmarshalled[0], {{if $this.Bytes}}{{else}}*{{end}}p.{{.Path}}) {
				{{end}}
					panic(fmt.Errorf("unmarshalled %v != p.{{.Path}} %v", unmarshalled[0], {{if $this.Bytes}}{{else}}*{{end}}p.{{.Path}}))
				}
				return
			}
			if len(unmarshalled) > 0 {
			{{if $this.Bytes}}
				t.Fatalf("Expected nil")
			{{else}}
				if unmarshalled[0] != *fieldpath.TestDefault("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}").GetDefault{{$this.CGoTyp}}() {
					t.Fatalf("wrong default")
				}
			{{end}}
			}
		{{end}}
	}

	{{if .Single}}

	type sorter{{.Struct}}{{$this.CName}} []*test.{{.Struct}}

	func (s sorter{{.Struct}}{{$this.CName}}) Less(i, j int) bool {
		{{if $this.Bytes}}
			if !({{.NotNil "s[j]"}}) {
				return false
			}
			if !({{.NotNil "s[i]"}}) {
				return true
			}
			return bytes.Compare(s[i].{{.Path}}, s[j].{{.Path}}) == -1
		{{else}}
		var vi {{$this.ReturnType}}
		var vj {{$this.ReturnType}}
		if ({{.NotNil "s[i]"}}) {
			{{if .Enum}}
			v1 := int32(*s[i].{{.Path}})
			vi = &v1
			{{else}}
			vi = s[i].{{.Path}}
			{{end}}
		} else {
			vi = fieldpath.TestDefault("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}").GetDefault{{$this.CGoTyp}}()
		}
		if ({{.NotNil "s[j]"}}) {
			{{if .Enum}}
			v2 := int32(*s[j].{{.Path}})
			vj = &v2
			{{else}}
			vj = s[j].{{.Path}}
			{{end}}
		} else {
			vj = fieldpath.TestDefault("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}").GetDefault{{$this.CGoTyp}}()
		}
		if vj == nil {
			return false
		}
		if vi == nil {
			return true
		}
		return *vi < *vj
		{{end}}
	}

	func (this sorter{{.Struct}}{{$this.CName}}) Len() int {
		return len(this)
	}

	func (this sorter{{.Struct}}{{$this.CName}}) Swap(i, j int) {
		this[i], this[j] = this[j], this[i]
	}

	func TestSort{{.Struct}}{{$this.CName}}(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}
		r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
		l := r.Intn(1000)
		unmarshalled := make(sorter{{.Struct}}{{$this.CName}}, l)
		marshalled := make([][]byte, l)
		var err error
		for i := 0; i < l; i++ {
			unmarshalled[i] = test.NewPopulated{{.Struct}}(r, false)
			marshalled[i], err = proto.Marshal(unmarshalled[i])
			if err != nil {
				panic(err)
			}
		}
		sort.Sort(unmarshalled)
		fp, err := fieldpath.New{{$this.CName}}SinglePath("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
		if err != nil {
			panic(err)
		}
		fpsorter := fieldpath.New{{$this.CName}}Sorter(&sortable{marshalled}, fp)
		sort.Sort(fpsorter)
		struct2 := &test.{{.Struct}}{}
		for i := 0; i < l; i++ {
			err = proto.Unmarshal(marshalled[i], struct2)
			if err != nil {
				panic(err)
			}
			if err := unmarshalled[i].VerboseEqual(struct2); err != nil {
				panic(err)
			}
		}
	}

	/* BENCHMARKS DISABLED

	func BenchmarkSortPath{{.Struct}}{{$this.CName}}(b *testing.B) {
		fp, err := fieldpath.New{{$this.CName}}SinglePath("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
		if err != nil {
			panic(err)
		}
		r := math_rand.New(math_rand.NewSource(pseudo))
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			l := r.Intn(1000)
			structs := make(sorter{{.Struct}}{{$this.CName}}, l)
			bytes := make([][]byte, l)
			for i := 0; i < l; i++ {
				structs[i] = test.NewPopulated{{.Struct}}(r, false)
				bytes[i], err = proto.Marshal(structs[i])
				if err != nil {
					panic(err)
				}
			}
			b.StartTimer()
			fpsorter := fieldpath.New{{$this.CName}}Sorter(&sortable{bytes}, fp)
			sort.Sort(fpsorter)
		}
	}

	func BenchmarkUnmarshalAndSort{{.Struct}}{{$this.CName}}(b *testing.B) {
		var err error
		r := math_rand.New(math_rand.NewSource(pseudo))
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			l := r.Intn(1000)
			structs := make(sort{{.Struct}}{{$this.CName}}, l)
			bytes := make([][]byte, l)
			for i := 0; i < l; i++ {
				structs[i] = test.NewPopulated{{.Struct}}(r, false)
				bytes[i], err = proto.Marshal(structs[i])
				if err != nil {
					panic(err)
				}
			}
			b.StartTimer()
			structs2 := make(sort{{.Struct}}{{$this.CName}}, l)
			for i := 0; i < l; i++ {
				s := &test.{{.Struct}}{}
				proto.Unmarshal(bytes[i], s)
				structs2[i] = s
			}
			sort.Sort(structs2)
		}
	}

	func BenchmarkJustSort{{.Struct}}{{$this.CName}}(b *testing.B) {
		r := math_rand.New(math_rand.NewSource(pseudo))
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			l := r.Intn(1000)
			structs := make(sort{{.Struct}}{{$this.CName}}, l)
			for i := 0; i < l; i++ {
				structs[i] = test.NewPopulated{{.Struct}}(r, false)
			}
			b.StartTimer()
			sort.Sort(structs)
		}
	}

	func BenchmarkUnmarshalPath{{.Struct}}{{$this.CName}}(b *testing.B) {
		r := math_rand.New(math_rand.NewSource(pseudo))
		fp, err := fieldpath.New{{$this.CName}}SinglePath("test", "{{.Struct}}", test.ThetestDescription(), "{{.Path}}")
		if err != nil {
			panic(err)
		}
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			p := test.NewPopulated{{.Struct}}(r, false)
			data, err := proto.Marshal(p)
			if err != nil {
				panic(err)
			}
			b.StartTimer()
			fp.Unmarshal(data)
		}
	}

	BENCHMARKS DISABLED */

	{{end}}

{{end}}
	`
}
