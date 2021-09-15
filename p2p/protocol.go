//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"io"

	"github.com/markkurossi/mpc/ot"
)

const (
	writeBufSize = 512 * 1024
	readBufSize  = 1024 * 1024
)

// Conn implements a protocol connection.
type Conn struct {
	closer    io.Closer
	writeBuf  []byte
	writePos  int
	writer    io.ReadWriter
	readBuf   []byte
	readStart int
	readEnd   int
	reader    io.ReadWriter
	Stats     IOStats
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
		closer:   closer,
		writeBuf: make([]byte, writeBufSize),
		readBuf:  make([]byte, readBufSize),
		writer:   conn,
		reader:   conn,
	}
}

// Flush flushed any pending data in the connection.
func (c *Conn) Flush() error {
	if c.writePos > 0 {
		n, err := c.writer.Write(c.writeBuf[0:c.writePos])
		c.Stats.Sent += uint64(n)
		c.writePos = 0
		return err
	}
	return nil
}

// Fill fills the input buffer from the connection. Any unused data in
// the buffer is moved to the beginning of the buffer.
func (c *Conn) Fill(n int) error {
	if c.readStart < c.readEnd {
		copy(c.readBuf[0:], c.readBuf[c.readStart:c.readEnd])
		c.readEnd -= c.readStart
		c.readStart = 0
	} else {
		c.readStart = 0
		c.readEnd = 0
	}
	for c.readStart+n > c.readEnd {
		got, err := c.reader.Read(c.readBuf[c.readEnd:])
		if err != nil {
			return err
		}
		c.Stats.Recvd += uint64(got)
		c.readEnd += got
	}
	return nil
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
	if c.writePos+1 > len(c.writeBuf) {
		if err := c.Flush(); err != nil {
			return err
		}
	}
	c.writeBuf[c.writePos] = val
	c.writePos++
	return nil
}

// SendUint16 sends an uint16 value.
func (c *Conn) SendUint16(val int) error {
	if c.writePos+2 > len(c.writeBuf) {
		if err := c.Flush(); err != nil {
			return err
		}
	}
	c.writeBuf[c.writePos+0] = byte((uint32(val) >> 8) & 0xff)
	c.writeBuf[c.writePos+1] = byte(uint32(val) & 0xff)
	c.writePos += 2
	return nil
}

// SendUint32 sends an uint32 value.
func (c *Conn) SendUint32(val int) error {
	if c.writePos+4 > len(c.writeBuf) {
		if err := c.Flush(); err != nil {
			return err
		}
	}
	c.writeBuf[c.writePos+0] = byte((uint32(val) >> 24) & 0xff)
	c.writeBuf[c.writePos+1] = byte((uint32(val) >> 16) & 0xff)
	c.writeBuf[c.writePos+2] = byte((uint32(val) >> 8) & 0xff)
	c.writeBuf[c.writePos+3] = byte(uint32(val) & 0xff)
	c.writePos += 4
	return nil
}

// SendData sends binary data.
func (c *Conn) SendData(val []byte) error {
	if c.writePos+4+len(val) > len(c.writeBuf) {
		if err := c.Flush(); err != nil {
			return err
		}
	}
	err := c.SendUint32(len(val))
	if err != nil {
		return err
	}
	copy(c.writeBuf[c.writePos:], val)
	c.writePos += len(val)
	return nil
}

// SendLabel sends an OT label.
func (c *Conn) SendLabel(val ot.Label, data *ot.LabelData) error {
	bytes := val.Bytes(data)
	if c.writePos+len(bytes) > len(c.writeBuf) {
		if err := c.Flush(); err != nil {
			return err
		}
	}
	copy(c.writeBuf[c.writePos:], bytes)
	c.writePos += len(bytes)

	return nil
}

// SendString sends a string value.
func (c *Conn) SendString(val string) error {
	return c.SendData([]byte(val))
}

// ReceiveByte receives a byte value.
func (c *Conn) ReceiveByte() (byte, error) {
	if c.readStart+1 > c.readEnd {
		if err := c.Fill(1); err != nil {
			return 0, err
		}
	}
	val := c.readBuf[c.readStart]
	c.readStart++
	return val, nil
}

// ReceiveUint16 receives an uint16 value.
func (c *Conn) ReceiveUint16() (int, error) {
	if c.readStart+2 > c.readEnd {
		if err := c.Fill(2); err != nil {
			return 0, err
		}
	}
	val := uint32(c.readBuf[c.readStart+0])
	val <<= 8
	val |= uint32(c.readBuf[c.readStart+1])
	c.readStart += 2

	return int(val), nil
}

// ReceiveUint32 receives an uint32 value.
func (c *Conn) ReceiveUint32() (int, error) {
	if c.readStart+4 > c.readEnd {
		if err := c.Fill(4); err != nil {
			return 0, err
		}
	}
	val := uint32(c.readBuf[c.readStart+0])
	val <<= 8
	val |= uint32(c.readBuf[c.readStart+1])
	val <<= 8
	val |= uint32(c.readBuf[c.readStart+2])
	val <<= 8
	val |= uint32(c.readBuf[c.readStart+3])
	c.readStart += 4

	return int(val), nil
}

// ReceiveData receives binary data.
func (c *Conn) ReceiveData() ([]byte, error) {
	len, err := c.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	if c.readStart+len > c.readEnd {
		if err := c.Fill(len); err != nil {
			return nil, err
		}
	}

	result := make([]byte, len)
	copy(result, c.readBuf[c.readStart:c.readStart+len])
	c.readStart += len

	return result, nil
}

// ReceiveLabel receives an OT label.
func (c *Conn) ReceiveLabel(val *ot.Label, data *ot.LabelData) error {
	if c.readStart+len(data) > c.readEnd {
		if err := c.Fill(len(data)); err != nil {
			return err
		}
	}
	copy(data[:], c.readBuf[c.readStart:c.readStart+len(data)])
	c.readStart += len(data)

	val.SetData(data)
	return nil
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
