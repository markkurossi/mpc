//
// pipe.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.

package ot

import (
	"fmt"
	"io"
)

var (
	_ IO = &Pipe{}
)

// Pipe implements the IO interface with in-memory io.Pipe.
type Pipe struct {
	rBuf []byte
	wBuf []byte
	r    *io.PipeReader
	w    *io.PipeWriter
}

// NewPipe creates a new in-memory pipe.
func NewPipe() (*Pipe, *Pipe) {
	ar, aw := io.Pipe()
	br, bw := io.Pipe()

	return &Pipe{
			rBuf: make([]byte, 64*1024),
			wBuf: make([]byte, 64*1024),
			r:    ar,
			w:    bw,
		}, &Pipe{
			rBuf: make([]byte, 64*1024),
			wBuf: make([]byte, 64*1024),
			r:    br,
			w:    aw,
		}
}

// SendData sends binary data.
func (p *Pipe) SendData(val []byte) error {
	l := len(val)
	bo.PutUint32(p.wBuf, uint32(l))
	n := copy(p.wBuf[4:], val)
	if n != l {
		return fmt.Errorf("pipe buffer too short: %d > %d", l, len(p.wBuf))
	}
	_, err := p.w.Write(p.wBuf[:4+l])
	return err
}

// SendUint32 sends an uint32 value.
func (p *Pipe) SendUint32(val int) error {
	bo.PutUint32(p.wBuf, uint32(val))
	_, err := p.w.Write(p.wBuf[:4])
	return err
}

// Flush flushed any pending data in the connection.
func (p *Pipe) Flush() error {
	return nil
}

// Drain consumes all input from the pipe.
func (p *Pipe) Drain() error {
	_, err := io.Copy(io.Discard, p.r)
	return err
}

// Close closes the pipe.
func (p *Pipe) Close() error {
	return p.w.Close()
}

// ReceiveData receives binary data.
func (p *Pipe) ReceiveData() ([]byte, error) {
	_, err := p.r.Read(p.rBuf[:4])
	if err != nil {
		return nil, err
	}
	l := bo.Uint32(p.rBuf)
	if l > uint32(len(p.rBuf)) {
		return nil, fmt.Errorf("pipe buffer too short: %d > %d", l, len(p.rBuf))
	}
	n, err := p.r.Read(p.rBuf[:])
	return p.rBuf[:n], err
}

// ReceiveUint32 receives an uint32 value.
func (p *Pipe) ReceiveUint32() (int, error) {
	_, err := p.r.Read(p.rBuf[:4])
	if err != nil {
		return 0, err
	}
	return int(bo.Uint32(p.rBuf)), nil
}
