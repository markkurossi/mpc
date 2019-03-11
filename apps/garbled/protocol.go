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
	"net"
)

func sendUint32(conn net.Conn, val int) {
	err := binary.Write(conn, binary.BigEndian, uint32(val))
	if err != nil {
		panic(err)
	}
}

func sendData(conn net.Conn, val []byte) {
	sendUint32(conn, len(val))
	_, err := conn.Write(val)
	if err != nil {
		panic(err)
	}
}

func receiveUint32(conn net.Conn) int {
	var buf [4]byte

	_, err := io.ReadFull(conn, buf[:])
	if err != nil {
		panic(err)
	}

	return int(binary.BigEndian.Uint32(buf[:]))
}

func receiveData(conn net.Conn) []byte {
	len := receiveUint32(conn)

	result := make([]byte, len)
	_, err := io.ReadFull(conn, result)
	if err != nil {
		panic(err)
	}

	return result
}
