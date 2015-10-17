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
)

type GenericPath interface {
	//Combines a path with the handler to create an Unmarshaler.
	NewUnmarshaler(h handler) *Unmarshaler
	//Returns whether the Field Type is a message.
	IsMessage() bool
}

type genericPath struct {
	*fdesc
}

//This constructor also checks if the path is valid and the type in the descriptor is matches the called function.
//This function should preferably be called in the init of a module, since this is not very type safe (stringly typed), this will help in catching these errors sooner rather than later.
//This is strictly less type safe than using a New<T>Path, but sometimes it cannot be helped, use New<T>Path when it can.
func NewGenericPath(rootPackage string, rootMessage string, descSet *descriptor.FileDescriptorSet, path string) (GenericPath, error) {
	fd, err := new(rootPackage, rootMessage, descSet, path)
	if err != nil {
		return nil, err
	}
	return &genericPath{
		fd,
	}, nil
}

//Returns whether the Field Type is a message.
func (this *genericPath) IsMessage() bool {
	return this.field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE
}

//Combines a path with the handler to create an Unmarshaler.
func (this *genericPath) NewUnmarshaler(h handler) *Unmarshaler {
	if !this.field.IsPacked() {
		switch this.field.GetType() {
		case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
			return newFloat64Unmarshaler(this.path, this.GetDefaultFloat64(), h)
		case descriptor.FieldDescriptorProto_TYPE_FLOAT:
			return newFloat32Unmarshaler(this.path, this.GetDefaultFloat32(), h)
		case descriptor.FieldDescriptorProto_TYPE_INT64:
			return newInt64Unmarshaler(this.path, this.GetDefaultInt64(), h)
		case descriptor.FieldDescriptorProto_TYPE_UINT64:
			return newUint64Unmarshaler(this.path, this.GetDefaultUint64(), h)
		case descriptor.FieldDescriptorProto_TYPE_INT32:
			return newInt32Unmarshaler(this.path, this.GetDefaultInt32(), h)
		case descriptor.FieldDescriptorProto_TYPE_FIXED64:
			return newFixed64Unmarshaler(this.path, this.GetDefaultUint64(), h)
		case descriptor.FieldDescriptorProto_TYPE_FIXED32:
			return newFixed32Unmarshaler(this.path, this.GetDefaultUint32(), h)
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			return newBoolUnmarshaler(this.path, this.GetDefaultBool(), h)
		case descriptor.FieldDescriptorProto_TYPE_STRING:
			return newStringUnmarshaler(this.path, this.GetDefaultString(), h)
		case descriptor.FieldDescriptorProto_TYPE_GROUP:
			panic("not implemented")
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			return newBytesUnmarshaler(this.path, nil, h)
		case descriptor.FieldDescriptorProto_TYPE_BYTES:
			return newBytesUnmarshaler(this.path, nil, h)
		case descriptor.FieldDescriptorProto_TYPE_UINT32:
			return newUint32Unmarshaler(this.path, this.GetDefaultUint32(), h)
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			return newInt32Unmarshaler(this.path, this.GetDefaultInt32(), h)
		case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
			return newSfixed32Unmarshaler(this.path, this.GetDefaultInt32(), h)
		case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
			return newSfixed64Unmarshaler(this.path, this.GetDefaultInt64(), h)
		case descriptor.FieldDescriptorProto_TYPE_SINT32:
			return newSint32Unmarshaler(this.path, this.GetDefaultInt32(), h)
		case descriptor.FieldDescriptorProto_TYPE_SINT64:
			return newSint64Unmarshaler(this.path, this.GetDefaultInt64(), h)
		}
		panic("unreachable")
	}
	switch this.field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return newPackedFloat64Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return newPackedFloat32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return newPackedInt64Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return newPackedUint64Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return newPackedInt32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return newPackedFixed64Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return newPackedFixed32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return newPackedBoolUnmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return newPackedStringUnmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		panic("unreachable: groups can't be packed")
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		panic("unreachable: messages can't be packed")
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return newPackedBytesUnmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return newPackedUint32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return newPackedInt32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return newPackedSfixed32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return newPackedSfixed64Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return newPackedSint32Unmarshaler(this.path, nil, h)
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return newPackedSint64Unmarshaler(this.path, nil, h)
	}
	panic("unreachable")
}
