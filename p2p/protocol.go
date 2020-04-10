//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/markkurossi/mpc/ot"
)

type Conn struct {
	closer io.Closer
	io     *bufio.ReadWriter
	Stats  IOStats
}

type IOStats struct {
	Sent  uint64
	Recvd uint64
}

func (stats IOStats) Add(o IOStats) IOStats {
	return IOStats{
		Sent:  stats.Sent + o.Sent,
		Recvd: stats.Recvd + o.Recvd,
	}
}

func (stats IOStats) Sub(o IOStats) IOStats {
	return IOStats{
		Sent:  stats.Sent - o.Sent,
		Recvd: stats.Recvd - o.Recvd,
	}
}

func (stats IOStats) Sum() uint64 {
	return stats.Sent + stats.Recvd
}

func NewConn(conn io.ReadWriter) *Conn {
	closer, _ := conn.(io.Closer)

	return &Conn{
		closer: closer,
		io: bufio.NewReadWriter(bufio.NewReader(conn),
			bufio.NewWriter(conn)),
	}
}

func (c *Conn) Flush() error {
	return c.io.Flush()
}

func (c *Conn) Close() error {
	if err := c.Flush(); err != nil {
		return err
	}
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}

func (c *Conn) SendByte(val byte) error {
	err := c.io.WriteByte(val)
	if err != nil {
		return err
	}
	c.Stats.Sent += 1
	return nil
}

func (c *Conn) SendUint32(val int) error {
	err := binary.Write(c.io, binary.BigEndian, uint32(val))
	if err != nil {
		return err
	}
	c.Stats.Sent += 4
	return nil
}

func (c *Conn) SendData(val []byte) error {
	err := c.SendUint32(len(val))
	if err != nil {
		return err
	}
	_, err = c.io.Write(val)
	if err != nil {
		return err
	}
	c.Stats.Sent += uint64(len(val))
	return nil
}

func (c *Conn) SendLabel(val ot.Label) error {
	n, err := c.io.Write(val.Bytes())
	if err != nil {
		return err
	}
	c.Stats.Sent += uint64(n)
	return nil
}

func (c *Conn) SendString(val string) error {
	return c.SendData([]byte(val))
}

func (c *Conn) ReceiveByte() (byte, error) {
	val, err := c.io.ReadByte()
	if err != nil {
		return 0, err
	}
	c.Stats.Recvd += 1
	return val, nil
}

func (c *Conn) ReceiveUint32() (int, error) {
	var buf [4]byte

	_, err := io.ReadFull(c.io, buf[:])
	if err != nil {
		return 0, err
	}
	c.Stats.Recvd += 4

	return int(binary.BigEndian.Uint32(buf[:])), nil
}

func (c *Conn) ReceiveData() ([]byte, error) {
	len, err := c.ReceiveUint32()
	if err != nil {
		return nil, err
	}

	result := make([]byte, len)
	_, err = io.ReadFull(c.io, result)
	if err != nil {
		return nil, err
	}
	c.Stats.Recvd += uint64(len)

	return result, nil
}

func (c *Conn) ReceiveLabel() (ot.Label, error) {
	var buf ot.LabelData
	n, err := io.ReadFull(c.io, buf[:])
	if err != nil {
		return ot.Label{}, err
	}
	c.Stats.Recvd += uint64(n)

	var result ot.Label
	result.SetData(&buf)
	return result, nil
}

func (c *Conn) ReceiveString() (string, error) {
	data, err := c.ReceiveData()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Conn) Receive(receiver *ot.Receiver, wire, bit uint) ([]byte, error) {

	if err := c.SendUint32(int(wire)); err != nil {
		return nil, err
	}
	c.Flush()

	xfer, err := receiver.NewTransfer(bit)
	if err != nil {
		return nil, err
	}

	x0, err := c.ReceiveData()
	if err != nil {
		return nil, err
	}
	x1, err := c.ReceiveData()
	if err != nil {
		return nil, err
	}
	err = xfer.ReceiveRandomMessages(x0, x1)
	if err != nil {
		return nil, err
	}

	v := xfer.V()
	if err := c.SendData(v); err != nil {
		return nil, err
	}
	c.Flush()

	m0p, err := c.ReceiveData()
	if err != nil {
		return nil, err
	}
	m1p, err := c.ReceiveData()
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
