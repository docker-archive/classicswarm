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

package fieldpath

import (
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strconv"
)

type def interface {
	GetDefaultFloat64() *float64
	GetDefaultFloat32() *float32
	GetDefaultInt32() *int32
	GetDefaultInt64() *int64
	GetDefaultUint64() *uint64
	GetDefaultUint32() *uint32
	GetDefaultBool() *bool
	GetDefaultString() *string
}

func (this *fdesc) GetDefaultFloat64() *float64 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var f2 float64
	if len(this.field.GetDefaultValue()) == 0 {
		return &f2
	}
	f, err := strconv.ParseFloat(this.field.GetDefaultValue(), 64)
	if err != nil {
		return &f2
	}
	return &f
}

func (this *fdesc) GetDefaultFloat32() *float32 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var f2 float32
	if len(this.field.GetDefaultValue()) == 0 {
		return &f2
	}
	f, err := strconv.ParseFloat(this.field.GetDefaultValue(), 32)
	if err != nil {
		return &f2
	}
	f2 = float32(f)
	return &f2
}

func (this *fdesc) GetDefaultInt32() *int32 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var v2 int32
	if len(this.field.GetDefaultValue()) == 0 {
		if this.field.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
			return &this.firstEnumValue
		}
		return &v2
	}
	v, err := strconv.ParseInt(this.field.GetDefaultValue(), 0, 32)
	if err != nil {
		return &v2
	}
	v2 = int32(v)
	return &v2
}

func (this *fdesc) GetDefaultInt64() *int64 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var v2 int64
	if len(this.field.GetDefaultValue()) == 0 {
		return &v2
	}
	v, err := strconv.ParseInt(this.field.GetDefaultValue(), 0, 64)
	if err != nil {
		return &v2
	}
	return &v
}

func (this *fdesc) GetDefaultUint64() *uint64 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var v2 uint64
	if len(this.field.GetDefaultValue()) == 0 {
		return &v2
	}
	v, err := strconv.ParseUint(this.field.GetDefaultValue(), 0, 64)
	if err != nil {
		return &v2
	}
	return &v
}

func (this *fdesc) GetDefaultUint32() *uint32 {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var v2 uint32
	if len(this.field.GetDefaultValue()) == 0 {
		return nil
	}
	v, err := strconv.ParseUint(this.field.GetDefaultValue(), 0, 32)
	if err != nil {
		return &v2
	}
	v2 = uint32(v)
	return &v2
}

func (this *fdesc) GetDefaultBool() *bool {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	var v2 bool
	if len(this.field.GetDefaultValue()) == 0 {
		return &v2
	}
	var v bool
	if this.field.GetDefaultValue() == "1" {
		v = true
	}
	return &v
}

func (this *fdesc) GetDefaultString() *string {
	if this.field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return nil
	}
	s := this.field.GetDefaultValue()
	return &s
}

//only for testing
func TestDefault(rootPackage string, rootMessage string, descSet *descriptor.FileDescriptorSet, path string) def {
	fd, err := new(rootPackage, rootMessage, descSet, path)
	if err != nil {
		panic(err)
	}
	return fd
}
