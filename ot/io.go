//
// io.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.

package ot

import (
	"math/big"
)

// IO defines an I/O interface to communicate between peers.
type IO interface {
	// SendData sends binary data.
	SendData(val []byte) error

	// SendUint32 sends an uint32 value.
	SendUint32(val int) error

	// Flush flushed any pending data in the connection.
	Flush() error

	// ReceiveData receives binary data.
	ReceiveData() ([]byte, error)

	// ReceiveUint32 receives an uint32 value.
	ReceiveUint32() (int, error)
}

// ReceiveBigInt receives a bit.Int from the connection.
func ReceiveBigInt(io IO) (*big.Int, error) {
	data, err := io.ReceiveData()
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(data), nil
}
