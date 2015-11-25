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
	"strings"
)

type errMerge struct {
	fieldName string
}

func (this *errMerge) Error() string {
	return this.fieldName + " requires merging"
}

// The NoMerge function checks that the marshaled protocol buffer does not require any merging when unmarshaling.
// When this property holds, streaming processing is possible.
//
// See below quotes from the protocol buffer documentation that describes how merging should work.
//
// https://developers.google.com/protocol-buffers/docs/encoding#optional
//
// Normally, an encoded message would never have more than one instance of an optional or required field.
// However, parsers are expected to handle the case in which they do.
// For numeric types and strings, if the same value appears multiple times, the parser accepts the last value it sees.
// For embedded message fields, the parser merges multiple instances of the same field,
// as if with the Message::MergeFrom method – that is,
// all singular scalar fields in the latter instance replace those in the former,
// singular embedded messages are merged, and repeated fields are concatenated.
// The effect of these rules is that parsing the concatenation of two encoded messages produces
// exactly the same result as if you had parsed the two messages separately and merged the resulting objects.
// That is, this:
//
// MyMessage message;
// message.ParseFromString(str1 + str2);
//
// is equivalent to this:
//
// MyMessage message, message2;
// message.ParseFromString(str1);
// message2.ParseFromString(str2);
// message.MergeFrom(message2);
//
// This property is occasionally useful, as it allows you to merge two messages even if you do not know their types.
//
// https://developers.google.com/protocol-buffers/docs/encoding#order
//
// However, protocol buffer parsers must be able to parse fields in any order,
// as not all messages are created by simply serializing an object – for instance,
// it's sometimes useful to merge two messages by simply concatenating them.
func NoMerge(buf []byte, descriptorSet *descriptor.FileDescriptorSet, rootPkg string, rootMsg string) error {
	msg := descriptorSet.GetMessage(rootPkg, rootMsg)
	if msg == nil {
		return &errUndefined{rootPkg, rootMsg, ""}
	}
	i := 0
	seen := make(map[int32]bool)
	for i < len(buf) {
		key, n, err := decodeVarint(buf, i)
		if err != nil {
			return err
		}
		i = i + n
		fieldNum := int32(key >> 3)
		wireType := int(key & 0x7)
		field := getFieldNumber(descriptorSet, rootPkg, rootMsg, msg, fieldNum)
		if field == nil || field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			i, err = skip(buf, i, wireType)
			if err != nil {
				return err
			}
			continue
		}
		if seen[fieldNum] {
			return &errMerge{msg.GetName() + "." + field.GetName()}
		}
		seen[fieldNum] = true
		if !field.IsMessage() {
			i, err = skip(buf, i, wireType)
			if err != nil {
				return err
			}
			continue
		}
		length, n, err := decodeVarint(buf, i)
		if err != nil {
			return err
		}
		i += n
		fieldPkg, fieldMsg := descriptorSet.FindMessage(rootPkg, rootMsg, field.GetName())
		if len(fieldMsg) == 0 {
			return &errUndefined{rootPkg, rootMsg, field.GetName()}
		}
		err = NoMerge(buf[i:i+int(length)], descriptorSet, fieldPkg, fieldMsg)
		if err != nil {
			return err
		}
		i += int(length)
	}
	return nil
}

func dotToUnderscore(r rune) rune {
	if r == '.' {
		return '_'
	}
	return r
}

func getFieldNumber(descriptorSet *descriptor.FileDescriptorSet, rootPkg string, rootMsg string, msg *descriptor.DescriptorProto, fieldNum int32) *descriptor.FieldDescriptorProto {
	for _, f := range msg.GetField() {
		if f.GetNumber() == fieldNum {
			return f
		}
	}
	if !msg.IsExtendable() {
		return nil
	}
	extendee := "." + rootPkg + "." + rootMsg
	for _, file := range descriptorSet.GetFile() {
		for _, ext := range file.GetExtension() {
			if strings.Map(dotToUnderscore, file.GetPackage()) == strings.Map(dotToUnderscore, rootPkg) {
				if !(ext.GetExtendee() == rootMsg || ext.GetExtendee() == extendee) {
					continue
				}
			} else {
				if ext.GetExtendee() != extendee {
					continue
				}
			}
			if ext.GetNumber() == fieldNum {
				return ext
			}
		}
	}
	return nil
}
