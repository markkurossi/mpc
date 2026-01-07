//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

/*

This implementation is derived from the EMP Toolkit's ikmp.h and cot.h
(https://github.com/emp-toolkit/emp-ot/blob/master/emp-ot/{ikmp,cot}.h)
with original license as follows:

MIT License

Copyright (c) 2018 Xiao Wang (wangxiao1254@gmail.com)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

Enquiries about further applications and development opportunities are welcome.

*/

package ot

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
)

const (
	// K defines the IKNP security parameter; the number of IKNP base
	// OTs.
	K = 128

	// Chunk size. Must be multiple of 16 (K-bits).
	chunkSize = 2 * 1024

	// The maximum number of byte-rows in a chunk.
	chunkByteRows = chunkSize / K

	// The number of label rows in a chunk.
	chunkRows = chunkByteRows * 8
)

// IKNPSender implements the random correlated OT sender.
type IKNPSender struct {
	// Delta defines the correlation delta: b1 = b0 ⊕ Δ
	Delta Label
	io    IO
	k0    [K]Label // XXX is this needed here?
	g0    [K]cipher.Stream
}

// NewIKNPSender creates a new sender. The d is an optional delta. If
// unset, the function creates a random delta.
func NewIKNPSender(base OT, io IO, r io.Reader, d *Label) (*IKNPSender, error) {
	var delta Label
	var err error
	if d == nil {
		delta, err = NewLabel(r)
		if err != nil {
			return nil, err
		}
	} else {
		delta = *d
	}

	s := &IKNPSender{
		Delta: delta,
		io:    io,
	}

	var flags [K]bool
	for i := 0; i < K; i++ {
		flags[i] = delta.Bit(i) == 1
	}

	err = base.Receive(flags[:], s.k0[:])
	if err != nil {
		return nil, err
	}

	var iv [16]byte
	var key LabelData

	for i := 0; i < K; i++ {
		block, err := aes.NewCipher(s.k0[i].Bytes(&key))
		if err != nil {
			return nil, err
		}
		s.g0[i] = cipher.NewCTR(block, iv[:])
	}

	return s, nil
}

// Send sends n labels. The function returns the b0 labels. The b1
// labels are b0[i] ⊕ s.Delta.
func (s *IKNPSender) Send(n int) ([]Label, error) {
	result := make([]Label, n)

	// The receiver sends the K*n-byte columns.
	var ofs int
	for ofs < n {
		chunk, err := s.io.ReceiveData()
		if err != nil {
			return nil, err
		}
		if len(chunk)%K != 0 {
			return nil, fmt.Errorf("invalid chunk size: %v", len(chunk))
		}
		byteRows := len(chunk) / K

		var t [chunkSize]byte

		for i := 0; i < K; i++ {
			prg(s.g0[i], t[i*byteRows:(i+1)*byteRows])
			if s.Delta.Bit(i) == 1 {
				xor(t[i*byteRows:(i+1)*byteRows], chunk[i*byteRows:])
			}
		}
		createLabels(result[ofs:], t[:], byteRows)

		ofs += byteRows * 8
	}

	return result, nil
}

// IKNPReceiver implements the random correlated OT receiver.
type IKNPReceiver struct {
	io IO
	g0 [K]cipher.Stream
	g1 [K]cipher.Stream
}

// NewIKNPReceiver creates a new receiver.
func NewIKNPReceiver(base OT, io IO, rand io.Reader) (*IKNPReceiver, error) {
	var wires [K]Wire
	for i := 0; i < K; i++ {
		l0, err := NewLabel(rand)
		if err != nil {
			return nil, err
		}
		l1, err := NewLabel(rand)
		if err != nil {
			return nil, err
		}
		wires[i] = Wire{
			L0: l0,
			L1: l1,
		}
	}
	err := base.Send(wires[:])
	if err != nil {
		return nil, err
	}

	r := &IKNPReceiver{
		io: io,
	}

	var key LabelData
	var iv [16]byte

	for i := 0; i < K; i++ {
		block, err := aes.NewCipher(wires[i].L0.Bytes(&key))
		if err != nil {
			return nil, err
		}
		r.g0[i] = cipher.NewCTR(block, iv[:])

		block, err = aes.NewCipher(wires[i].L1.Bytes(&key))
		if err != nil {
			return nil, err
		}
		r.g1[i] = cipher.NewCTR(block, iv[:])
	}

	return r, nil
}

// Receive labels based on the selection flags b. The returned labels
// implement the correlation: br[i] = b0[i] ⊕ b[i]*s.Delta.
func (r *IKNPReceiver) Receive(b []bool) ([]Label, error) {
	bbuf := make([]byte, (len(b)+7)/8)
	for i, f := range b {
		if f {
			bbuf[i/8] |= 1 << (i % 8)
		}
	}

	result := make([]Label, len(b))

	var chunk, out [chunkSize]byte
	var tmp [chunkByteRows]byte

	for ofs := 0; ofs < len(b); {
		rows := chunkRows
		avail := len(b) - ofs
		if rows > avail {
			rows = avail
		}
		byteRows := (rows + 7) / 8

		for i := 0; i < K; i++ {
			prg(r.g0[i], chunk[i*byteRows:(i+1)*byteRows])
			prg(r.g1[i], tmp[:byteRows])

			xor(tmp[:byteRows], chunk[i*byteRows:])
			xor(tmp[:byteRows], bbuf[ofs/8:])

			copy(out[i*byteRows:], tmp[:byteRows])
		}
		if err := r.io.SendData(out[:byteRows*128]); err != nil {
			return nil, err
		}
		if err := r.io.Flush(); err != nil {
			return nil, err
		}
		createLabels(result[ofs:], chunk[:], byteRows)

		ofs += rows
	}

	return result, nil
}

func prg(c cipher.Stream, buf []byte) {
	// Clear buffer as it is shared between different caller's
	// iterations.
	for i := 0; i < len(buf); i++ {
		buf[i] = 0
	}
	c.XORKeyStream(buf, buf)
}

func createLabels(l []Label, buf []byte, w int) {
	end := w * 8
	if end > len(l) {
		end = len(l)
	}
	for i := 0; i < end; i++ {
		row := i / 8
		bit := i % 8
		for j := 0; j < K; j++ {
			v := uint((buf[j*w+row] >> bit) & 1)
			l[i].SetBit(j, v)
		}
	}
}
