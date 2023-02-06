//
// encryption_block.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//
// PKCS #1 Encryption-block formatting, RFC 2313.

package pkcs1

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// EncryptionBlockType specifies the encryption block type and how the
// padding is detected.
type EncryptionBlockType byte

// Block types.
const (
	BT0 EncryptionBlockType = iota
	BT1
	BT2
)

const (
	// MinPadLen specifies the minimum padding length.
	MinPadLen = 8
)

var (
	// ErrorInvalidEncryptionBlock error is returned in the encryption
	// block is malformed.
	ErrorInvalidEncryptionBlock = errors.New("invalid encryption block")
)

// NewEncryptionBlock creates a new encryption block with the given
// type and data. The argument blockLen specifies the length of the
// resulting block. The function will return an error if the blockLen
// is too long to contain valid block formatting and MinPadLen of
// padding. A block type BT, a padding string PS, and the data D shall
// be formatted into an octet string EB, the encryption block.
//
//	EB = 00 || BT || PS || 00 || D .           (1)
func NewEncryptionBlock(bt EncryptionBlockType, blockLen int, data []byte) (
	[]byte, error) {

	padLen := blockLen - 3 - len(data)
	if padLen < MinPadLen {
		return nil, errors.New("data too long")
	}

	block := make([]byte, blockLen)
	block[0] = 0
	block[1] = byte(bt)

	switch bt {
	case BT0:
		return nil, errors.New("block type 0 not supported")

	case BT1:
		for i := 0; i < padLen; i++ {
			block[2+i] = 0xff
		}

	case BT2:
		_, err := io.ReadFull(rand.Reader, block[2:padLen+2])
		if err != nil {
			return nil, err
		}
		for i := 0; i < padLen; i++ {
			for block[2+i] == 0 {
				if _, err := rand.Read(block[2+i : 2+i+1]); err != nil {
					return nil, err
				}
			}
		}
	}
	copy(block[3+padLen:], data)

	return block, nil
}

// ParseEncryptionBlock parses the argument encryption block and
// returns its data.
func ParseEncryptionBlock(block []byte) ([]byte, error) {
	if len(block) < 4 {
		return nil, errors.New("truncated encryption block")
	}
	if block[0] != 0 {
		return nil, ErrorInvalidEncryptionBlock
	}
	switch EncryptionBlockType(block[1]) {
	case BT1, BT2:
	default:
		return nil, fmt.Errorf("invalid encryption block type %d", block[1])
	}

	for i := 2; i < len(block); i++ {
		if block[i] == 0 {
			return block[i+1:], nil
		}
	}
	return nil, ErrorInvalidEncryptionBlock
}
