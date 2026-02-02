//
// co.go
//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

/*

This implementation is derived from the EMP Toolkit's cot.h
(https://github.com/emp-toolkit/emp-ot/blob/master/emp-ot/cot.h)
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
	"fmt"
	"io"
)

// ROT implements random OT as the OT interface.
type ROT struct {
	base      OT
	r         io.Reader
	malicious bool
	shared    bool
	io        IO
	iknpS     *IKNPSender
	iknpR     *IKNPReceiver
}

// NewROT creates an IKNP-based Random OT instance implementing the OT
// interface.
//
// The malicious flag selects the adversary model. If malicious is
// true, the implementation enables KOS-style consistency checks and
// provides security against malicious adversaries (with abort). If
// false, the protocol reduces to semi-honest IKNP.
//
// The shared flag controls whether the ROT instance may be shared
// across multiple sessions. If shared is true, the ROT is initialized
// only once on the first call to InitSender or InitResponder.
// Subsequent initialization calls are ignored and MUST use the same
// role as the initial one.
func NewROT(base OT, r io.Reader, malicious, shared bool) *ROT {
	return &ROT{
		base:      base,
		r:         r,
		malicious: malicious,
		shared:    shared,
	}
}

// InitSender implements OT.InitSender.
func (rot *ROT) InitSender(io IO) error {
	if rot.iknpR != nil {
		return fmt.Errorf("already initialized as receiver")
	}
	if rot.iknpS != nil {
		if !rot.shared {
			return fmt.Errorf("already initialized")
		}
		// Ensure sender and receiver are in sync.
		return rot.io.Flush()
	}
	err := rot.base.InitSender(io)
	if err != nil {
		return err
	}
	s, err := NewIKNPSender(rot.base, io, rot.r, nil)
	if err != nil {
		return err
	}
	rot.io = io
	rot.iknpS = s

	return nil
}

// InitReceiver implements OT.InitReceiver.
func (rot *ROT) InitReceiver(io IO) error {
	if rot.iknpS != nil {
		return fmt.Errorf("already initialized as sender")
	}
	if rot.iknpR != nil {
		if !rot.shared {
			return fmt.Errorf("already initialized")
		}
		// Ensure sender and receiver are in sync.
		return rot.io.Flush()
	}
	err := rot.base.InitReceiver(io)
	if err != nil {
		return err
	}
	s, err := NewIKNPReceiver(rot.base, io, rot.r)
	if err != nil {
		return err
	}
	rot.io = io
	rot.iknpR = s

	return nil
}

// Send implements OT.Send.
func (rot *ROT) Send(wires []Wire) error {
	if rot.iknpS == nil {
		return fmt.Errorf("not initialized as sender")
	}
	data, err := rot.iknpS.Send(len(wires), rot.malicious)
	if err != nil {
		return err
	}
	seed, err := NewLabel(rot.r)
	if err != nil {
		return err
	}
	mitccrh := NewMITCCRH(seed, otBatchSize)

	var ld LabelData
	err = rot.io.SendLabel(seed, &ld)
	if err != nil {
		return err
	}
	err = rot.io.Flush()
	if err != nil {
		return err
	}

	pad := make([]Label, 2*otBatchSize)
	for i := 0; i < len(wires); i += otBatchSize {
		end := i + otBatchSize
		if end > len(wires) {
			end = len(wires)
		}
		for j := i; j < end; j++ {
			pad[2*(j-i)] = data[j]
			pad[2*(j-i)+1] = data[j]
			pad[2*(j-i)+1].Xor(rot.iknpS.Delta)
		}
		mitccrh.Hash(pad, otBatchSize, 2)
		for j := i; j < end; j++ {
			wires[j].L0 = pad[2*(j-i)]
			wires[j].L1 = pad[2*(j-i)+1]
		}
	}

	return nil
}

// Receive implements OT.Receive.
func (rot *ROT) Receive(flags []bool, result []Label) error {
	if rot.iknpR == nil {
		return fmt.Errorf("not initialized as receiver")
	}
	err := rot.iknpR.Receive(flags, result, rot.malicious)
	if err != nil {
		return err
	}
	var seed Label
	var ld LabelData
	err = rot.io.ReceiveLabel(&seed, &ld)
	if err != nil {
		return err
	}
	mitccrh := NewMITCCRH(seed, otBatchSize)

	pad := make([]Label, otBatchSize)
	for i := 0; i < len(flags); i += otBatchSize {
		copy(pad, result[i:])
		mitccrh.Hash(pad, otBatchSize, 1)
		copy(result[i:], pad)
	}

	return nil
}
