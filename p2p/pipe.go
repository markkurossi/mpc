//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"io"
)

// Pipe implements the Conn interface as a bidirectional communication
// pipe. Anything send to the first endpoint can be received from the
// second and vice versa.
func Pipe() (*Conn, *Conn) {
	var p0, p1 pipe

	p0.r, p1.w = io.Pipe()
	p1.r, p0.w = io.Pipe()

	return NewConn(&p0), NewConn(&p1)
}

type pipe struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (p *pipe) Close() error {
	if err := p.r.Close(); err != nil {
		return err
	}
	return p.w.Close()
}

func (p *pipe) Read(data []byte) (n int, err error) {
	return p.r.Read(data)
}

func (p *pipe) Write(data []byte) (n int, err error) {
	return p.w.Write(data)
}
