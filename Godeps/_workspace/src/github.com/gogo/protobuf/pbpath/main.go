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
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/gogo/protobuf/fieldpath"
	"github.com/gogo/protobuf/parser"
	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

var protoPath string
var protoFilename string
var descFilename string
var inputDelim string
var debugFlag bool
var expandFlag bool

func init() {
	protoPathUsage := "same as protoc --proto_path"
	protoPathDefault := "."
	flag.StringVar(&protoPath, "proto_path", protoPathDefault, protoPathUsage)
	protoFilenameUsage := "proto filename containing the description of the protocol buffers"
	protoFilenameDefault := ""
	flag.StringVar(&protoFilename, "proto_filename", protoFilenameDefault, protoFilenameUsage)
	descFilenameUsage := "parsed descriptor_set_out, alternative to proto_filename"
	descFilenameDefault := ""
	flag.StringVar(&descFilename, "descriptor_set_in", descFilenameDefault, descFilenameUsage)
	flag.StringVar(&descFilename, "D", descFilenameDefault, descFilenameUsage+" (shorthand)")
	inputDelimUsage := "input length delimeter (none|varint|big32)"
	inputDelimDefault := "none"
	flag.StringVar(&inputDelim, "length_in", inputDelimDefault, inputDelimUsage)
	flag.StringVar(&inputDelim, "i", inputDelimDefault, inputDelimUsage+" (shorthand)")
	flag.BoolVar(&debugFlag, "debug", false, "debug flag")
	expandUsage := "set if the fieldpaths are collapsed w.r.t. go embedding and needs to be expanded to be compatible with protocol buffer fieldpath"
	flag.BoolVar(&expandFlag, "expand", false, expandUsage)
	flag.BoolVar(&expandFlag, "e", false, expandUsage+" (shorthand)")
	flag.Parse()
}

func er(s string) {
	os.Stderr.Write([]byte("ERROR: " + s + "\n"))
	if debugFlag {
		panic(s)
	}
	os.Exit(1)
}

type ToStdout struct {
	BytesFunc func(v []byte)
}

func (*ToStdout) Float64(v float64) {
	os.Stdout.Write([]byte(fmt.Sprintf("%f\n", v)))
}

func (*ToStdout) Float32(v float32) {
	os.Stdout.Write([]byte(fmt.Sprintf("%f\n", v)))
}

func (*ToStdout) Int64(v int64) {
	os.Stdout.Write([]byte(fmt.Sprintf("%d\n", v)))
}

func (*ToStdout) Int32(v int32) {
	os.Stdout.Write([]byte(fmt.Sprintf("%d\n", v)))
}

func (*ToStdout) Uint64(v uint64) {
	os.Stdout.Write([]byte(fmt.Sprintf("%d\n", v)))
}

func (*ToStdout) Uint32(v uint32) {
	os.Stdout.Write([]byte(fmt.Sprintf("%d\n", v)))
}

func (*ToStdout) Bool(v bool) {
	os.Stdout.Write([]byte(fmt.Sprintf("%t\n", v)))
}

func (this *ToStdout) Bytes(v []byte) {
	this.BytesFunc(v)
}

func (*ToStdout) String(v string) {
	os.Stdout.Write([]byte(v + "\n"))
}

type reader interface {
	Read() []byte
}

type varintReader struct {
	r   *bufio.Reader
	ten []byte
}

func newVarintReader(r io.Reader) *varintReader {
	return &varintReader{bufio.NewReader(r), make([]byte, 10)}
}

func (this *varintReader) Read() []byte {
	v, err := binary.ReadUvarint(this.r)
	if err != nil {
		er(err.Error())
	}
	l := int(v)
	data := make([]byte, l)
	_, err = io.ReadFull(this.r, data)
	if err != nil {
		er(err.Error())
	}
	return data
}

type big32Reader struct {
	r io.Reader
}

func newBig32Reader(r io.Reader) *big32Reader {
	return &big32Reader{r}
}

func (this *big32Reader) Read() []byte {
	l := uint32(0)
	if err := binary.Read(this.r, binary.BigEndian, &l); err != nil {
		if err == io.EOF {
			return nil
		}
		er(err.Error())
	}
	data := make([]byte, l)
	_, err := io.ReadFull(this.r, data)
	if err != nil {
		er(err.Error())
	}
	return data
}

type allReader struct {
	r io.Reader
}

func newAllReader(r io.Reader) *allReader {
	return &allReader{r}
}

func (this *allReader) Read() []byte {
	data, err := ioutil.ReadAll(this.r)
	if err != nil {
		er(err.Error())
	}
	return data
}

func main() {
	if len(flag.Args()) == 0 {
		er("fieldpath needs to be provided")
	}
	var desc = &descriptor.FileDescriptorSet{}
	if len(descFilename) > 0 {
		data, err := ioutil.ReadFile(descFilename)
		if err != nil {
			er(err.Error())
		}
		if err := proto.Unmarshal(data, desc); err != nil {
			er(fmt.Sprintf("Reading descriptor_set_in filename (%v) : %v", descFilename, err))
		}
	}
	if len(protoFilename) > 0 {
		var err error
		desc, err = parser.ParseFile(protoFilename, strings.Split(protoPath, ":")...)
		if err != nil {
			er(fmt.Sprintf("Parsing proto filename (%v) with proto path (%v) : %v", protoFilename, protoPath, err))
		}
	}
	if desc == nil {
		er("either descriptor_set_in or proto_filename flag needs to be provided")
	}

	us := []*fieldpath.Unmarshaler{}
	printers := []func(v []byte){}
	for _, a := range flag.Args() {
		fieldpaths := strings.Split(a, ".")
		if len(fieldpaths) < 2 {
			er("fieldpath flag needs at least a package.message structure")
		}
		rootPkg := fieldpaths[0]
		rootMsg := fieldpaths[1]
		if len(fieldpaths) >= 3 {
			fpath := strings.Join(fieldpaths[2:], ".")
			if expandFlag {
				var err error
				fpath, err = fieldpath.Expand(rootPkg, rootMsg, fpath, desc)
				if err != nil {
					er(err.Error())
				}
			}
			field, err := fieldpath.NewGenericPath(rootPkg, rootMsg, desc, fpath)
			if err != nil {
				er(err.Error())
			}
			stdout := &ToStdout{}
			stdout.BytesFunc = func(v []byte) {
				os.Stdout.Write([]byte(fmt.Sprintf("%v\n", v)))
			}
			if field.IsMessage() {
				stdout.BytesFunc = func(v []byte) {
					err := fieldpath.ToString(rootPkg, rootMsg, desc, fpath, v, 0, os.Stdout)
					if err != nil {
						er(err.Error())
					}
				}
			}
			u := field.NewUnmarshaler(stdout)
			us = append(us, u)
		} else {
			printers = append(printers, func(v []byte) {
				err := fieldpath.ToString(rootPkg, rootMsg, desc, "", v, 0, os.Stdout)
				if err != nil {
					er(err.Error())
				}
			})
		}

	}

	var r reader
	if inputDelim == "varint" {
		r = newVarintReader(os.Stdin)
	} else if inputDelim == "big32" {
		r = newBig32Reader(os.Stdin)
	} else {
		r = newAllReader(os.Stdin)
	}

	c := fieldpath.Compile(us...)

	for {
		data := r.Read()
		if data == nil {
			return
		}
		for _, p := range printers {
			p(data)
		}
		err := c.Unmarshal(data)
		if err != nil {
			er(err.Error())
		}
		if inputDelim == "none" {
			return
		}
	}

}
