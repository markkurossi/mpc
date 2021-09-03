//
// Copyright (c) 2019-2021 Markku Rossi
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

// Conn implements a protocol connection.
type Conn struct {
	closer io.Closer
	io     *bufio.ReadWriter
	Stats  IOStats
}

// IOStats implements I/O statistics.
type IOStats struct {
	Sent  uint64
	Recvd uint64
}

// Add adds the argument stats to this IOStats and returns the sum.
func (stats IOStats) Add(o IOStats) IOStats {
	return IOStats{
		Sent:  stats.Sent + o.Sent,
		Recvd: stats.Recvd + o.Recvd,
	}
}

// Sub subtracts the argument stats from this IOStats and returns the
// difference.
func (stats IOStats) Sub(o IOStats) IOStats {
	return IOStats{
		Sent:  stats.Sent - o.Sent,
		Recvd: stats.Recvd - o.Recvd,
	}
}

// Sum returns sum of sent and received bytes.
func (stats IOStats) Sum() uint64 {
	return stats.Sent + stats.Recvd
}

// NewConn creates a new connection around the argument connection.
func NewConn(conn io.ReadWriter) *Conn {
	closer, _ := conn.(io.Closer)

	return &Conn{
		closer: closer,
		io: bufio.NewReadWriter(bufio.NewReader(conn),
			bufio.NewWriter(conn)),
	}
}

// Flush flushed any pending data in the connection.
func (c *Conn) Flush() error {
	return c.io.Flush()
}

// Close flushes any pending data and closes the connection.
func (c *Conn) Close() error {
	if err := c.Flush(); err != nil {
		return err
	}
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}

// SendByte sends a byte value.
func (c *Conn) SendByte(val byte) error {
	err := c.io.WriteByte(val)
	if err != nil {
		return err
	}
	c.Stats.Sent++
	return nil
}

// SendUint16 sends an uint16 value.
func (c *Conn) SendUint16(val int) error {
	if err := c.io.WriteByte(byte((uint32(val) >> 8) & 0xff)); err != nil {
		return err
	}
	if err := c.io.WriteByte(byte(uint32(val) & 0xff)); err != nil {
		return err
	}
	c.Stats.Sent += 2
	return nil
}

// SendUint32 sends an uint32 value.
func (c *Conn) SendUint32(val int) error {
	if err := c.io.WriteByte(byte((uint32(val) >> 24) & 0xff)); err != nil {
		return err
	}
	if err := c.io.WriteByte(byte((uint32(val) >> 16) & 0xff)); err != nil {
		return err
	}
	if err := c.io.WriteByte(byte((uint32(val) >> 8) & 0xff)); err != nil {
		return err
	}
	if err := c.io.WriteByte(byte(uint32(val) & 0xff)); err != nil {
		return err
	}
	c.Stats.Sent += 4
	return nil
}

// SendData sends binary data.
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

// SendLabel sends an OT label.
func (c *Conn) SendLabel(val ot.Label, data *ot.LabelData) error {
	n, err := c.io.Write(val.Bytes(data))
	if err != nil {
		return err
	}
	c.Stats.Sent += uint64(n)
	return nil
}

// SendString sends a string value.
func (c *Conn) SendString(val string) error {
	return c.SendData([]byte(val))
}

// ReceiveByte receives a byte value.
func (c *Conn) ReceiveByte() (byte, error) {
	val, err := c.io.ReadByte()
	if err != nil {
		return 0, err
	}
	c.Stats.Recvd++
	return val, nil
}

// ReceiveUint16 receives an uint16 value.
func (c *Conn) ReceiveUint16() (int, error) {
	var buf [2]byte

	_, err := io.ReadFull(c.io, buf[:])
	if err != nil {
		return 0, err
	}
	c.Stats.Recvd += 2

	return int(binary.BigEndian.Uint16(buf[:])), nil
}

// ReceiveUint32 receives an uint32 value.
func (c *Conn) ReceiveUint32() (int, error) {
	var buf [4]byte

	_, err := io.ReadFull(c.io, buf[:])
	if err != nil {
		return 0, err
	}
	c.Stats.Recvd += 4

	return int(binary.BigEndian.Uint32(buf[:])), nil
}

// ReceiveData receives binary data.
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

// ReceiveLabel receives an OT label.
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

// ReceiveString receives a string value.
func (c *Conn) ReceiveString() (string, error) {
	data, err := c.ReceiveData()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Receive implements OT receive for the bit value of a wire.
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
