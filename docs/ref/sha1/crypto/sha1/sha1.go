// -*- go -*-
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sha1 implements the SHA-1 hash algorithm as defined in RFC 3174.
//
// SHA-1 is cryptographically broken and should not be used for secure
// applications.
package sha1

import (
	"encoding/binary"
)

// The size of a SHA-1 checksum in bytes.
const Size = 20

// The blocksize of SHA-1 in bytes.
const BlockSize = 64

const (
	chunk = 64
	init0 = 0x67452301
	init1 = 0xEFCDAB89
	init2 = 0x98BADCFE
	init3 = 0x10325476
	init4 = 0xC3D2E1F0
)

// Sum returns the SHA-1 checksum of the data.
func Sum(data []byte) [Size]byte {
	state := [5]uint32{init0, init1, init2, init3, init4}
	length := uint64(len(data))

	for len(data) >= chunk {
		state = Block(data[:chunk], state)
		data = data[chunk:]
	}

	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.

	var tmp [128 + 8]byte // padding + length buffer
	copy(tmp[:], data)
	i := len(data)
	tmp[i] = 0x80

	var t uint64
	if length%64 < 56 {
		t = 56 - length%64
	} else {
		t = 64 + 56 - length%64
	}
	t += uint64(i)

	// Length in bits.
	length <<= 3
	padlen := tmp[:t+8]
	binary.BigEndian.PutUint64(padlen[t:], length)

	for len(padlen) >= chunk {
		state = Block(padlen[:chunk], state)
		padlen = padlen[chunk:]
	}

	var digest [Size]byte

	binary.BigEndian.PutUint32(digest[0:], state[0])
	binary.BigEndian.PutUint32(digest[4:], state[1])
	binary.BigEndian.PutUint32(digest[8:], state[2])
	binary.BigEndian.PutUint32(digest[12:], state[3])
	binary.BigEndian.PutUint32(digest[16:], state[4])

	return digest
}
