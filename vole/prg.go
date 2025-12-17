//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package vole

import (
	"crypto/aes"
	"crypto/cipher"
)

// prgExpandLabel is a deterministic PRF. Callers must ensure domain
// separation via unique keys.
func prgExpandLabel(key [16]byte, out *[32]byte) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		panic(err)
	}

	var iv [16]byte
	stream := cipher.NewCTR(block, iv[:])

	var zero [32]byte
	stream.XORKeyStream(out[:], zero[:])
}
