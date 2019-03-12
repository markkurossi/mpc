//
// protocol.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/binary"
	"io"
)

func sendUint32(conn io.Writer, val int) error {
	return binary.Write(conn, binary.BigEndian, uint32(val))
}

func sendData(conn io.Writer, val []byte) error {
	err := sendUint32(conn, len(val))
	if err != nil {
		return err
	}
	_, err = conn.Write(val)
	return err
}

func receiveUint32(conn io.Reader) (int, error) {
	var buf [4]byte

	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint32(buf[:])), nil
}

func receiveData(conn io.Reader) ([]byte, error) {
	len, err := receiveUint32(conn)
	if err != nil {
		return nil, err
	}

	result := make([]byte, len)
	_, err = io.ReadFull(conn, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
