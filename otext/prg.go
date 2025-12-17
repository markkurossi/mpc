//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package otext

import (
	"crypto/aes"
	"crypto/cipher"
)

func prgAESCTR(key []byte, out []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	var iv [16]byte
	stream := cipher.NewCTR(block, iv[:])

	stream.XORKeyStream(out[:], out[:])
}
