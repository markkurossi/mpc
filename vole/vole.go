//
// Copyright (c) 2025-2026 Markku Rossi
//
// All rights reserved.
//

package vole

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

type Sender struct {
	oti  ot.OT
	conn *p2p.Conn
	iknp *ot.IKNPSender
}

func NewSender(oti ot.OT, conn *p2p.Conn, r io.Reader) (*Sender, error) {
	e := &Sender{
		oti:  oti,
		conn: conn,
	}
	if err := e.oti.InitSender(e.conn); err != nil {
		return nil, err
	}

	var err error
	e.iknp, err = ot.NewIKNPSender(oti, conn, r, nil)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Sender) Mul(inputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	m := len(inputs)
	if m == 0 {
		return nil, nil
	}

	// Packed path: Expand(m) -> wires (one wire per triple)
	labels, err := e.iknp.Send(m, false)
	if err != nil {
		return nil, fmt.Errorf("vole: ExpandSend: %w", err)
	}
	if len(labels) != m {
		return nil, fmt.Errorf("vole: ExpandSend returned %d wires, want %d", len(labels), m)
	}

	// Derive r_i from L0 label using PRG and reduce mod p.
	rs := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		var ld ot.LabelData
		labels[i].GetData(&ld)

		// Expand label to 32-bytes.
		var pad [32]byte
		prgExpandLabel(ld, &pad)

		rsi := new(big.Int).SetBytes(pad[:])
		rsi.Mod(rsi, p)
		rs[i] = rsi
	}

	// Now receive the receiver's packed y-vector (m*32 bytes).
	// (This is the single extra message; we still benefit from 1 wire/triple.)
	yb, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: MulSender receive y-vector: %w", err)
	}
	if len(yb) != m*32 {
		return nil, fmt.Errorf("vole: MulSender expected %d bytes for y-vector, got %d", m*32, len(yb))
	}
	ys := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		ys[i] = new(big.Int).SetBytes(yb[off : off+32])
		ys[i].Mod(ys[i], p)
	}

	// compute u_i = r_i + x_i * y_i mod p; send packed u vector back.
	out := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		tmp := new(big.Int).Mul(inputs[i], ys[i])
		tmp.Mod(tmp, p)
		ui := new(big.Int).Add(rs[i], tmp)
		ui.Mod(ui, p)
		out = append(out, bytes32(ui)...)
	}

	if err := e.conn.SendData(out); err != nil {
		return nil, fmt.Errorf("vole: MulSender send u-vector: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: MulSender flush u-vector: %w", err)
	}

	return rs, nil
}

type Receiver struct {
	oti  ot.OT
	conn *p2p.Conn
	iknp *ot.IKNPReceiver
}

func NewReceiver(oti ot.OT, conn *p2p.Conn, r io.Reader) (*Receiver, error) {
	e := &Receiver{
		oti:  oti,
		conn: conn,
	}
	if err := e.oti.InitReceiver(e.conn); err != nil {
		return nil, err
	}

	var err error
	e.iknp, err = ot.NewIKNPReceiver(oti, conn, r)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// Mul functionality: inputs = y[0..m-1]. Returns u[0..m-1] = r_i + x_i * y_i.
func (e *Receiver) Mul(inputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	if e == nil {
		return nil, errors.New("vole: nil Ext")
	}
	m := len(inputs)
	if m == 0 {
		return nil, nil
	}

	// Packed receiver path:
	// ExpandReceive with dummy flags (we only need labels; true packed removal of the extra
	// message would require a more complex linear mapping).
	flags := make([]bool, m)
	for i := 0; i < m; i++ {
		flags[i] = false // we could encode something useful here in a future refinement
	}

	labels := make([]ot.Label, m)

	err := e.iknp.Receive(flags, labels, false)
	if err != nil {
		return nil, fmt.Errorf("vole: ExpandReceive: %w", err)
	}
	if len(labels) != m {
		return nil, fmt.Errorf("vole: ExpandReceive returned %d labels, want %d", len(labels), m)
	}

	// Derive local pads (not strictly required in this design, but available)
	// and (crucially) send the packed y-vector to sender, receive u-vector back.
	outY := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		outY = append(outY, bytes32(inputs[i])...)
	}
	if err := e.conn.SendData(outY); err != nil {
		return nil, fmt.Errorf("vole: MulReceiver send y-vector: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: MulReceiver flush y-vector: %w", err)
	}

	// receive u vector
	ub, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: MulReceiver receive u-vector: %w", err)
	}
	if len(ub) != m*32 {
		return nil, fmt.Errorf("vole: MulReceiver expected %d bytes for u-vector, got %d", m*32, len(ub))
	}
	us := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		us[i] = new(big.Int).SetBytes(ub[off : off+32])
		us[i].Mod(us[i], p)
	}
	return us, nil
}

// -----------------------------------------------------------------------------
// small IO / helpers
// -----------------------------------------------------------------------------

func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}

func bytesToUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// randomFieldElementFromCrypto reduces 32 random bytes modulo p
func randomFieldElementFromCrypto(r io.Reader, p *big.Int) (*big.Int, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	x := new(big.Int).SetBytes(b)
	x.Mod(x, p)
	return x, nil
}

func bytes32(v *big.Int) []byte {
	out := make([]byte, 32)
	if v == nil {
		return out
	}
	b := v.Bytes()
	copy(out[32-len(b):], b)
	return out
}
