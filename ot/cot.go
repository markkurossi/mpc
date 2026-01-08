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

const (
	otBatchSize = 8
)

// COT implements IKNP OT as the OT interface.
type COT struct {
	base  OT
	r     io.Reader
	io    IO
	iknpS *IKNPSender
	iknpR *IKNPReceiver
}

// NewCOT creates a IKNP OT implementing the OT interface.
func NewCOT(base OT, r io.Reader) *COT {
	return &COT{
		base: base,
		r:    r,
	}
}

// InitSender implements OT.InitSender.
func (cot *COT) InitSender(io IO) error {
	if cot.iknpS != nil || cot.iknpR != nil {
		return fmt.Errorf("already initialized")
	}
	err := cot.base.InitSender(io)
	if err != nil {
		return err
	}
	s, err := NewIKNPSender(cot.base, io, cot.r, nil)
	if err != nil {
		return err
	}
	cot.io = io
	cot.iknpS = s

	return nil
}

// InitReceiver implements OT.InitReceiver.
func (cot *COT) InitReceiver(io IO) error {
	if cot.iknpS != nil || cot.iknpR != nil {
		return fmt.Errorf("already initialized")
	}
	err := cot.base.InitReceiver(io)
	if err != nil {
		return err
	}
	s, err := NewIKNPReceiver(cot.base, io, cot.r)
	if err != nil {
		return err
	}
	cot.io = io
	cot.iknpR = s

	return nil
}

// Send implements OT.Send.
func (cot *COT) Send(wires []Wire) error {
	if cot.iknpS == nil {
		return fmt.Errorf("not initialized as sender")
	}
	data, err := cot.iknpS.Send(len(wires))
	if err != nil {
		return err
	}
	seed, err := NewLabel(cot.r)
	if err != nil {
		return err
	}
	mitccrh := NewMITCCRH(seed, otBatchSize)

	var ld LabelData
	err = cot.io.SendLabel(seed, &ld)
	if err != nil {
		return err
	}
	err = cot.io.Flush()
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
			pad[2*(j-i)+1].Xor(cot.iknpS.Delta)
		}
		mitccrh.Hash(pad, otBatchSize, 2)
		for j := i; j < end; j++ {
			pad[2*(j-i)].Xor(wires[j].L0)
			pad[2*(j-i)+1].Xor(wires[j].L1)
		}
		for j := 0; j < 2*(end-i); j++ {
			err = cot.io.SendLabel(pad[j], &ld)
			if err != nil {
				return err
			}
		}
	}
	return cot.io.Flush()
}

// Receive implements OT.Receive.
func (cot *COT) Receive(flags []bool, result []Label) error {
	if cot.iknpR == nil {
		return fmt.Errorf("not initialized as receiver")
	}
	data, err := cot.iknpR.Receive(flags)
	if err != nil {
		return err
	}
	var seed Label
	var ld LabelData
	err = cot.io.ReceiveLabel(&seed, &ld)
	if err != nil {
		return err
	}
	mitccrh := NewMITCCRH(seed, otBatchSize)

	pad := make([]Label, otBatchSize)

	for i := 0; i < len(flags); i += otBatchSize {
		end := otBatchSize
		if end > len(flags)-i {
			end = len(flags) - i
		}
		for j := 0; j < end; j++ {
			pad[j] = data[i+j]
		}
		mitccrh.Hash(pad, otBatchSize, 1)

		var res0 Label
		var res1 Label

		for j := 0; j < end; j++ {
			err = cot.io.ReceiveLabel(&res0, &ld)
			if err != nil {
				return err
			}
			err = cot.io.ReceiveLabel(&res1, &ld)
			if err != nil {
				return err
			}
			if flags[i+j] {
				result[i+j] = res1
			} else {
				result[i+j] = res0
			}
			result[i+j].Xor(pad[j])
		}
	}

	return nil
}
