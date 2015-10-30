// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"errors"
	"io"
)

// A Decoder reads and decodes JSON objects from an input stream.
type Decoder struct {
	r    io.Reader
	buf  []byte
	d    decodeState
	scan scanner
	err  error
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
// Number instead of as a float64.
func (dec *Decoder) UseNumber() { dec.d.useNumber = true }

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of JSON into a Go value.
func (dec *Decoder) Decode(v interface{}) error {
	if dec.err != nil {
		return dec.err
	}

	n, err := dec.readValue()
	if err != nil {
		return err
	}

	// Don't save err from unmarshal into dec.err:
	// the connection is still usable since we read a complete JSON
	// object from it before the error happened.
	dec.d.init(dec.buf[0:n])
	err = dec.d.unmarshal(v)

	// Slide rest of data down.
	rest := copy(dec.buf, dec.buf[n:])
	dec.buf = dec.buf[0:rest]

	return err
}

// Buffered returns a reader of the data remaining in the Decoder's
// buffer. The reader is valid until the next call to Decode.
func (dec *Decoder) Buffered() io.Reader {
	return bytes.NewReader(dec.buf)
}

// readValue reads a JSON value into dec.buf.
// It returns the length of the encoding.
func (dec *Decoder) readValue() (int, error) {
	dec.scan.reset()

	scanp := 0
	var err error
Input:
	for {
		// Look in the buffer for a new value.
		for i, c := range dec.buf[scanp:] {
			dec.scan.bytes++
			v := dec.scan.step(&dec.scan, int(c))
			if v == scanEnd {
				scanp += i
				break Input
			}
			// scanEnd is delayed one byte.
			// We might block trying to get that byte from src,
			// so instead invent a space byte.
			if (v == scanEndObject || v == scanEndArray) && dec.scan.step(&dec.scan, ' ') == scanEnd {
				scanp += i + 1
				break Input
			}
			if v == scanError {
				dec.err = dec.scan.err
				return 0, dec.scan.err
			}
		}
		scanp = len(dec.buf)

		// Did the last read have an error?
		// Delayed until now to allow buffer scan.
		if err != nil {
			if err == io.EOF {
				if dec.scan.step(&dec.scan, ' ') == scanEnd {
					break Input
				}
				if nonSpace(dec.buf) {
					err = io.ErrUnexpectedEOF
				}
			}
			dec.err = err
			return 0, err
		}

		// Make room to read more into the buffer.
		const minRead = 512
		if cap(dec.buf)-len(dec.buf) < minRead {
			newBuf := make([]byte, len(dec.buf), 2*cap(dec.buf)+minRead)
			copy(newBuf, dec.buf)
			dec.buf = newBuf
		}

		// Read.  Delay error for next iteration (after scan).
		var n int
		n, err = dec.r.Read(dec.buf[len(dec.buf):cap(dec.buf)])
		dec.buf = dec.buf[0 : len(dec.buf)+n]
	}
	return scanp, nil
}

func nonSpace(b []byte) bool {
	for _, c := range b {
		if !isSpace(rune(c)) {
			return true
		}
	}
	return false
}

// An Encoder writes JSON objects to an output stream.
type Encoder struct {
	w         io.Writer
	err       error
	canonical bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Canonical causes the encoder to switch to Canonical JSON mode.
// Read more at: http://wiki.laptop.org/go/Canonical_JSON
func (enc *Encoder) Canonical() { enc.canonical = true }

// Encode writes the JSON encoding of v to the stream,
// followed by a newline character.
//
// See the documentation for Marshal for details about the
// conversion of Go values to JSON.
func (enc *Encoder) Encode(v interface{}) error {
	if enc.err != nil {
		return enc.err
	}
	e := newEncodeState(enc.canonical)
	err := e.marshal(v)
	if err != nil {
		return err
	}

	if !enc.canonical {
		// Terminate each value with a newline.
		// This makes the output look a little nicer
		// when debugging, and some kind of space
		// is required if the encoded value was a number,
		// so that the reader knows there aren't more
		// digits coming.
		e.WriteByte('\n')
	}

	if _, err = enc.w.Write(e.Bytes()); err != nil {
		enc.err = err
	}
	encodeStatePool.Put(e)
	return err
}

// RawMessage is a raw encoded JSON object.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
type RawMessage []byte

// MarshalJSON returns *m as the JSON encoding of m.
func (m *RawMessage) MarshalJSON() ([]byte, error) {
	return *m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

var _ Marshaler = (*RawMessage)(nil)
var _ Unmarshaler = (*RawMessage)(nil)
