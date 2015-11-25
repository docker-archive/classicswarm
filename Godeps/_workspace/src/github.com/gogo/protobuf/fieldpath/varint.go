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
	"io"
	"strconv"
)

func sizeOfVarint(buf []byte, offset int) (int, error) {
	var n int
	l := len(buf)
	for {
		if offset+n >= l {
			return 0, io.ErrUnexpectedEOF
		}
		if buf[offset+n] < 0x80 {
			break
		}
		n++
	}
	return n + 1, nil
}

func decodeVarint(buf []byte, offset int) (x uint64, n int, err error) {
	l := len(buf)
	for shift := uint(0); ; shift += 7 {
		if offset+n >= l {
			err = io.ErrUnexpectedEOF
			return
		}
		b := buf[offset+n]
		x |= (uint64(b) & 0x7F) << shift
		if b < 0x80 {
			break
		}
		n++
	}
	n += 1
	return
}

func skip(buf []byte, offset int, wireType int) (int, error) {
	switch wireType {
	case 0: //WireVarint:
		nn, err := sizeOfVarint(buf, offset)
		return offset + nn, err
	case 1: //WireFixed64:
		return offset + 8, nil
	case 2: //WireLengthDelimited:
		l, nn, err := decodeVarint(buf, offset)
		offset += nn + int(l)
		return offset, err
	case 5: //WireFixed32:
		return offset + 4, nil
	}
	panic("unreachable wireType = " + strconv.Itoa(wireType))
}
