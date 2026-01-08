//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//
// Better Concrete Security for Half-Gates Garbling (in the
// Multi-Instance Setting)
//  - https://eprint.iacr.org/2019/1168.pdf

/*

This implementation is derived from the EMP Toolkit's mitccrh.h
(https://github.com/emp-toolkit/emp-tool/blob/master/emp-tool/utils/mitccrh.h)
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
)

// MITCCRH implements the multi-instance TCCR hash (MITCCRH).
type MITCCRH struct {
	batchSize int

	startPoint Label
	gid        uint64

	ciphers []cipher.Block
	keyUsed int
}

// NewMITCCRH creates a new MITCCRH with the seed s and batchSize.
func NewMITCCRH(s Label, batchSize int) *MITCCRH {
	return &MITCCRH{
		batchSize:  batchSize,
		startPoint: s,
		ciphers:    make([]cipher.Block, batchSize),
		keyUsed:    batchSize, // force renew on first use
	}
}

func (m *MITCCRH) renewKeys() {
	for i := 0; i < m.batchSize; i++ {
		// Init key as tweak
		key := Label{
			D0: m.gid,
			D1: 0,
		}
		m.gid++

		key.Xor(m.startPoint)

		var d LabelData
		block, err := aes.NewCipher(key.Bytes(&d))
		if err != nil {
			panic(err)
		}
		m.ciphers[i] = block
	}
	m.keyUsed = 0
}

// Hash hashes k*h blocks. Each key k is used to hash h consecutive
// blocks in blks.
func (m *MITCCRH) Hash(blks []Label, k, h int) {
	if k > m.batchSize {
		panic("k > batchSize")
	}
	if m.batchSize%k != 0 {
		panic("batchSize % k != 0")
	}
	if k*h != len(blks) {
		panic("k*h != len(blks)")
	}
	if m.keyUsed == m.batchSize {
		m.renewKeys()
	}

	tmp := make([]LabelData, len(blks))
	for i := 0; i < len(blks); i++ {
		blks[i].GetData(&tmp[i])
	}

	// Use each key 1...k to encrypt h consecutive blocks.
	for i := 0; i < k; i++ {
		c := m.ciphers[m.keyUsed+i]
		for j := 0; j < h; j++ {
			idx := i*h + j
			c.Encrypt(tmp[idx][:], tmp[idx][:])
		}
	}
	m.keyUsed += k

	// XOR input with encrypted blocks.
	for i := range blks {
		var t Label
		t.SetData(&tmp[i])
		blks[i].Xor(t)
	}
}
