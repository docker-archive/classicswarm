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

package parser

import (
	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"os/exec"
	"strings"
)

type errCmd struct {
	output []byte
	err    error
}

func (this *errCmd) Error() string {
	return this.err.Error() + ":" + string(this.output)
}

func ParseFile(filename string, paths ...string) (*descriptor.FileDescriptorSet, error) {
	return parseFile(filename, false, true, paths...)
}

func parseFile(filename string, includeSourceInfo bool, includeImports bool, paths ...string) (*descriptor.FileDescriptorSet, error) {
	args := []string{"--proto_path=" + strings.Join(paths, ":")}
	if includeSourceInfo {
		args = append(args, "--include_source_info")
	}
	if includeImports {
		args = append(args, "--include_imports")
	}
	args = append(args, "--descriptor_set_out=/dev/stdout")
	args = append(args, filename)
	cmd := exec.Command("protoc", args...)
	cmd.Env = []string{}
	data, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &errCmd{data, err}
	}
	fileDesc := &descriptor.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileDesc); err != nil {
		return nil, err
	}
	return fileDesc, nil
}
