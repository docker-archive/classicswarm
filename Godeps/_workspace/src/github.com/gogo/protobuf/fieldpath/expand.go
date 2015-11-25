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
	"github.com/gogo/protobuf/gogoproto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strings"
)

type errUndefined struct {
	rootPkg string
	rootMsg string
	path    string
}

func (this *errUndefined) Error() string {
	return fmt.Sprintf("undefined field %s.%s.%s", this.rootPkg, this.rootMsg, this.path)
}

//Expands a fieldpath as defined in go with possible implications for embedded structs to a normal proto fieldpath.
func Expand(rootPkg string, rootMsg string, path string, descriptorSet *descriptor.FileDescriptorSet) (string, error) {
	msg := descriptorSet.GetMessage(rootPkg, rootMsg)
	if msg == nil {
		return "", &errUndefined{rootPkg, rootMsg, path}
	}
	paths := strings.Split(path, ".")
	if len(paths) == 0 {
		return "", &errUndefined{rootPkg, rootMsg, path}
	}
	for _, f := range msg.GetField() {
		if f.IsMessage() {
			newRootPkg, newRootMsg := descriptorSet.FindMessage(rootPkg, rootMsg, f.GetName())
			if len(newRootPkg) == 0 || len(newRootMsg) == 0 {
				return "", &errUndefined{rootPkg, rootMsg, path}
			}
			if gogoproto.IsEmbed(f) {
				newPath, err := Expand(newRootPkg, newRootMsg, strings.Join(paths, "."), descriptorSet)
				if err == nil {
					return f.GetName() + "." + newPath, nil
				}
			} else if paths[0] == f.GetName() {
				newPath, err := Expand(newRootPkg, newRootMsg, strings.Join(paths[1:], "."), descriptorSet)
				if err != nil {
					return "", err
				}
				return f.GetName() + "." + newPath, nil
			}
		} else if (len(paths) == 1) && (paths[0] == f.GetName()) {
			return path, nil
		}
	}
	if msg.IsExtendable() {
		newRootPkg, f := descriptorSet.FindExtension(rootPkg, rootMsg, paths[0])
		if f == nil {
			return "", &errUndefined{rootPkg, rootMsg, path}
		}
		typeName := f.GetTypeName()
		typeNames := strings.Split(typeName, ".")
		newRootMsg := typeName
		if len(typeNames) > 1 {
			newRootMsg = typeNames[len(typeNames)-1]
		}
		if f.IsMessage() {
			if gogoproto.IsEmbed(f) {
				newPath, err := Expand(newRootPkg, newRootMsg, strings.Join(paths, "."), descriptorSet)
				if err == nil {
					return f.GetName() + "." + newPath, nil
				}
			} else {
				newPath, err := Expand(newRootPkg, newRootMsg, strings.Join(paths[1:], "."), descriptorSet)
				if err != nil {
					return "", err
				}
				return f.GetName() + "." + newPath, nil
			}
		} else if (len(paths) == 1) && (paths[0] == f.GetName()) {
			return path, nil
		}
	}
	return "", &errUndefined{rootPkg, rootMsg, path}
}

//Collapses a proto fieldpath into a go fieldpath.  They are different if some of the fields in the fieldpath have been embedded.
func Collapse(rootPkg string, rootMsg string, path string, descriptorSet *descriptor.FileDescriptorSet) (string, error) {
	msg := descriptorSet.GetMessage(rootPkg, rootMsg)
	if msg == nil {
		return "", &errUndefined{rootPkg, rootMsg, path}
	}
	paths := strings.Split(path, ".")
	if len(paths) == 0 {
		return "", &errUndefined{rootPkg, rootMsg, path}
	}
	if len(paths) == 1 {
		return path, nil
	}
	for _, f := range msg.GetField() {
		if f.GetName() != paths[0] {
			continue
		}
		if f.IsMessage() {
			newRootPkg, newRootMsg := descriptorSet.FindMessage(rootPkg, rootMsg, f.GetName())
			if len(newRootPkg) == 0 || len(newRootMsg) == 0 {
				return "", &errUndefined{rootPkg, rootMsg, path}
			}
			newPath, err := Collapse(newRootPkg, newRootMsg, strings.Join(paths[1:], "."), descriptorSet)
			if err != nil {
				return "", err
			}
			if gogoproto.IsEmbed(f) {
				return newPath, nil
			} else {
				return paths[0] + "." + newPath, nil
			}
		}
	}
	if msg.IsExtendable() {
		newRootPkg, f := descriptorSet.FindExtension(rootPkg, rootMsg, paths[0])
		if f == nil {
			return "", &errUndefined{rootPkg, rootMsg, path}
		}
		typeName := f.GetTypeName()
		typeNames := strings.Split(typeName, ".")
		newRootMsg := typeName
		if len(typeNames) > 1 {
			newRootMsg = typeNames[len(typeNames)-1]
		}
		newPath, err := Collapse(newRootPkg, newRootMsg, strings.Join(paths[1:], "."), descriptorSet)
		if err != nil {
			return "", err
		}
		if gogoproto.IsEmbed(f) {
			return newPath, nil
		} else {
			return paths[0] + "." + newPath, nil
		}
	}
	return "", nil
}
