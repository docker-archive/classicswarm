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
	"fmt"
	"io"
	"strings"

	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type stringWriter struct {
	tabs int
	w    io.Writer
}

func (this *stringWriter) tab() {
	for i := 0; i < this.tabs; i++ {
		io.WriteString(this.w, "\t")
	}
}

func (this *stringWriter) Float64(v float64) {
	io.WriteString(this.w, fmt.Sprintf(`%f`, v))
}

func (this *stringWriter) Float32(v float32) {
	io.WriteString(this.w, fmt.Sprintf(`%f`, v))
}

func (this *stringWriter) Int64(v int64) {
	io.WriteString(this.w, fmt.Sprintf(`%d`, v))
}

func (this *stringWriter) Int32(v int32) {
	io.WriteString(this.w, fmt.Sprintf(`%d`, v))
}

func (this *stringWriter) Uint64(v uint64) {
	io.WriteString(this.w, fmt.Sprintf(`%d`, v))
}

func (this *stringWriter) Uint32(v uint32) {
	io.WriteString(this.w, fmt.Sprintf(`%d`, v))
}

func (this *stringWriter) Bool(v bool) {
	io.WriteString(this.w, fmt.Sprintf(`%v`, v))
}

func (this *stringWriter) Bytes(v []byte) {
	io.WriteString(this.w, fmt.Sprintf(`%v`, v))
}

func (this *stringWriter) String(v string) {
	io.WriteString(this.w, v)
}

//Converts a marshalled protocol buffer to string.
//It does this pretty effienctly by not combining slices.
//It rather prints every element of the slice seperately.
//This is typically used for debugging.
func ToString(rootPkg string, rootMsg string, desc *descriptor.FileDescriptorSet, fpath string, buf []byte, tabs int, writer io.Writer) error {
	w := &stringWriter{tabs, writer}
	messagePkg := rootPkg
	messageName := rootMsg
	if len(fpath) > 0 {
		fd, err := new(rootPkg, rootMsg, desc, fpath)
		if err != nil {
			panic(err)
		}
		messagePkg = fd.rootPkg
		messageName = fd.rootMsg
	}
	msg := desc.GetMessage(messagePkg, messageName)
	if msg == nil {
		panic("message not found: " + messagePkg + "." + messageName)
	}
	offset := 0
	for offset < len(buf) {
		key, n, err := decodeVarint(buf, offset)
		if err != nil {
			panic(err)
		}
		offset += n
		fieldNum := int32(key >> 3)
		found := false
		for _, f := range msg.GetField() {
			if f.IsPacked() {
				panic("not implemented")
			}
			if f.GetNumber() != fieldNum {
				continue
			}
			found = true
			if f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				length, nn, err := decodeVarint(buf, offset)
				if err != nil {
					panic(err)
				}
				offset += nn
				w.tab()
				w.String(f.GetName() + `:{` + "\n")
				err = ToString(messagePkg, messageName, desc, f.GetName(), buf[offset:offset+int(length)], tabs+1, writer)
				if err != nil {
					return err
				}
				offset += int(length)
				w.tab()
				w.String(`}` + "\n")
				break
			} else {
				w.tab()
				w.String(f.GetName() + `:`)
				var n int
				var err error
				switch f.GetType() {
				case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
					n, err = DecFloat64(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_FLOAT:
					n, err = DecFloat32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_INT64:
					n, err = DecInt64(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_UINT64:
					n, err = DecUint64(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_INT32:
					n, err = DecInt32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_FIXED64:
					n, err = DecFixed64(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_FIXED32:
					n, err = DecFixed32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_BOOL:
					n, err = DecBool(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_STRING:
					n, err = DecString(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_GROUP:
					panic("not implemented")
				case descriptor.FieldDescriptorProto_TYPE_BYTES:
					n, err = DecBytes(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_UINT32:
					n, err = DecUint32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_ENUM:
					n, err = DecInt32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
					n, err = DecSfixed32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
					n, err = DecSfixed64(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_SINT32:
					n, err = DecSint32(buf, offset, w)
				case descriptor.FieldDescriptorProto_TYPE_SINT64:
					n, err = DecSint64(buf, offset, w)
				default:
					panic("unreachable")
				}
				offset += n
				if err != nil {
					panic(err)
				}
				w.String(",\n")
				break
			}
		}
		if !found {
			for _, extRange := range msg.GetExtensionRange() {
				if extRange.GetStart() <= fieldNum && fieldNum <= extRange.GetEnd() {
					found = true
					break
				}
			}
			w.tab()
			offset -= n
			nn, err := protoSkip(buf[offset:])
			if err != nil {
				panic(err)
			}
			if !found {
				w.String("XXX_unrecognized:")
			} else {
				w.String("XXX_extensions:")
				extPkg, ext := desc.FindExtensionByFieldNumber(messagePkg, messageName, fieldNum)
				if ext != nil && ext.IsMessage() {
					names := strings.Split(ext.GetTypeName(), ".")
					if len(names) == 3 {
						_, nnn, err := decodeVarint(buf, offset+n)
						if err != nil {
							panic(err)
						}
						w.tab()
						w.String(extPkg + "." + names[2] + `:{` + "\n")
						err = ToString(extPkg, names[2], desc, "", buf[offset+n+nnn:offset+nn], tabs+1, writer)
						if err != nil {
							return err
						}
						offset += nn
						w.tab()
						w.String(`}` + "\n")
						break
					} else {
						panic(ext.GetTypeName())
					}
				}
			}
			w.Bytes(buf[offset : offset+nn])
			w.String("\n")
			offset += nn
		}
	}
	return nil
}

func protoSkip(data []byte) (n int, err error) {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for {
				if index >= l {
					return 0, io.ErrUnexpectedEOF
				}
				index++
				if data[index-1] < 0x80 {
					break
				}
			}
			return index, nil
		case 1:
			index += 8
			return index, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			index += length
			return index, nil
		case 3:
			for {
				var innerWire uint64
				var start int = index
				for shift := uint(0); ; shift += 7 {
					if index >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[index]
					index++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := protoSkip(data[start:])
				if err != nil {
					return 0, err
				}
				index = start + next
			}
			return index, nil
		case 4:
			return index, nil
		case 5:
			index += 4
			return index, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}
