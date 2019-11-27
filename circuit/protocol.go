//
// protocol.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bufio"
	"encoding/binary"
	"io"
	"math/big"

	"github.com/markkurossi/mpc/ot"
)

const (
	OP_OT = iota
	OP_RESULT
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

func receive(conn *bufio.ReadWriter, receiver *ot.Receiver, wire, bit int) (
	[]byte, error) {

	if err := sendUint32(conn, OP_OT); err != nil {
		return nil, err
	}
	if err := sendUint32(conn, wire); err != nil {
		return nil, err
	}
	conn.Flush()

	xfer, err := receiver.NewTransfer(bit)
	if err != nil {
		return nil, err
	}

	x0, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	x1, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	err = xfer.ReceiveRandomMessages(x0, x1)
	if err != nil {
		return nil, err
	}

	v := xfer.V()
	if err := sendData(conn, v); err != nil {
		return nil, err
	}
	conn.Flush()

	m0p, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	m1p, err := receiveData(conn)
	if err != nil {
		return nil, err
	}

	err = xfer.ReceiveMessages(m0p, m1p, nil)
	if err != nil {
		return nil, err
	}

	m, _ := xfer.Message()
	return m, nil
}

func result(conn *bufio.ReadWriter, labels []*ot.Label) (*big.Int, error) {
	if err := sendUint32(conn, OP_RESULT); err != nil {
		return nil, err
	}
	for _, l := range labels {
		if err := sendData(conn, l.Bytes()); err != nil {
			return nil, err
		}
	}
	conn.Flush()

	result, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(result), nil
}
