//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package otext

import (
	"errors"
	"io"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

const (
	// IKNPK defines the security parameter k of the IKNP
	// protocol. The IKNPK is the number of base-OTs.
	IKNPK = 128
)

// IKNPSender implements the sender side of the IKNP OT extension.
type IKNPSender struct {
	base    ot.OT
	conn    *p2p.Conn
	choices []bool
	seedS   []ot.LabelData
}

// IKNPReceiver implements the receiver side of the IKNP OT extension.
type IKNPReceiver struct {
	base  ot.OT
	conn  *p2p.Conn
	seed0 []ot.LabelData
	seed1 []ot.LabelData
}

// NewIKNPSender creates a new IKNP sender.
func NewIKNPSender(base ot.OT, conn *p2p.Conn, r io.Reader) (
	*IKNPSender, error) {

	// Base OT receiver: choose random choice bits and call
	// base.Receive.
	choices := make([]bool, IKNPK)
	var buf [IKNPK / 8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}
	for i := 0; i < IKNPK; i++ {
		choices[i] = ((buf[i/8] >> uint(i%8)) & 1) == 1
	}

	labels := make([]ot.Label, IKNPK)
	if err := base.Receive(choices, labels); err != nil {
		return nil, err
	}

	seedS := make([]ot.LabelData, IKNPK)
	for i := 0; i < IKNPK; i++ {
		labels[i].GetData(&seedS[i])
	}

	return &IKNPSender{
		base:    base,
		conn:    conn,
		choices: choices,
		seedS:   seedS,
	}, nil
}

// Expand implements the sender IKNP expansion.
func (iknp *IKNPSender) Expand(n int) ([]ot.Wire, error) {
	if n <= 0 {
		return nil, errors.New("n must be positive")
	}

	rowBytes := (n + 7) / 8
	total := IKNPK * rowBytes

	// Receive U (k xor-rows) from receiver: U = T0 ^ T1 for each row
	U, err := iknp.conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	if len(U) < total {
		return nil, errors.New("not enough U")
	}

	// Generate T0 rows from seedS and, if choice bit == 1, PRG(seedS)
	// ^ U_row would be T0?  The standard IKNP computation yields rows
	// equal to T0 for all i after this step.  We'll compute rows =
	// PRG(seedS) XOR (choices[i] ? U_row : 0) which yields T0 in
	// either case.
	rows := make([][]byte, IKNPK)
	for i := 0; i < IKNPK; i++ {
		rows[i] = make([]byte, rowBytes)
		prgAESCTR(iknp.seedS[i][:], rows[i])

		if iknp.choices[i] {
			urow := U[i*rowBytes : (i+1)*rowBytes]
			for j := 0; j < rowBytes; j++ {
				rows[i][j] ^= urow[j]
			}
		}
	}

	// Build wires. For each column j build:
	//  L0 from T0 (rows)
	//  L1 from T1 = T0 ^ U_row
	wires := make([]ot.Wire, n)
	for j := 0; j < n; j++ {
		var b0 ot.LabelData
		var b1 ot.LabelData

		for bit := 0; bit < 128; bit++ {
			if bit >= IKNPK {
				break
			}
			byteRow := j / 8
			bitPos := uint(j % 8)

			// T0 bit
			rowBit := (rows[bit][byteRow] >> bitPos) & 1
			bytePos := bit / 8
			inner := uint(7 - (bit % 8))
			if rowBit == 1 {
				b0[bytePos] |= (1 << inner)
			}

			// T1 bit = T0_bit ^ U_row_bit
			urow := U[bit*rowBytes : (bit+1)*rowBytes]
			uBit := (urow[byteRow] >> bitPos) & 1
			if (rowBit ^ uBit) == 1 {
				b1[bytePos] |= (1 << inner)
			}
		}

		var L0, L1 ot.Label

		L0.SetData(&b0)
		L1.SetData(&b1)

		wires[j] = ot.Wire{L0: L0, L1: L1}
	}

	return wires, nil
}

// NewIKNPReceiver creates a new IKNP receiver.
func NewIKNPReceiver(base ot.OT, conn *p2p.Conn, r io.Reader) (
	*IKNPReceiver, error) {

	// Base OT sender: prepare k label pairs and call base.Send.
	seed0 := make([]ot.LabelData, IKNPK)
	seed1 := make([]ot.LabelData, IKNPK)

	wires := make([]ot.Wire, IKNPK)
	for i := 0; i < IKNPK; i++ {
		l0, err := ot.NewLabel(r)
		if err != nil {
			return nil, err
		}
		l1, err := ot.NewLabel(r)
		if err != nil {
			return nil, err
		}
		l0.GetData(&seed0[i])
		l1.GetData(&seed1[i])

		wires[i] = ot.Wire{L0: l0, L1: l1}
	}
	if err := base.Send(wires); err != nil {
		return nil, err
	}

	return &IKNPReceiver{
		base:  base,
		conn:  conn,
		seed0: seed0,
		seed1: seed1,
	}, nil
}

// Expand implements the receiver IKNP expansion.
func (iknp *IKNPReceiver) Expand(flags []bool) ([]ot.Label, error) {
	N := len(flags)
	if N == 0 {
		return nil, errors.New("flags empty")
	}
	rowBytes := (N + 7) / 8

	T0 := make([][]byte, IKNPK)
	T1 := make([][]byte, IKNPK)
	for i := 0; i < IKNPK; i++ {
		T0[i] = make([]byte, rowBytes)
		T1[i] = make([]byte, rowBytes)
		prgAESCTR(iknp.seed0[i][:], T0[i])
		prgAESCTR(iknp.seed1[i][:], T1[i])
	}

	// Send U rows (T0 xor T1) concatenated
	U := make([]byte, IKNPK*rowBytes)
	for i := 0; i < IKNPK; i++ {
		for j := 0; j < rowBytes; j++ {
			U[i*rowBytes+j] = T0[i][j] ^ T1[i][j]
		}
	}
	if err := iknp.conn.SendData(U); err != nil {
		return nil, err
	}
	if err := iknp.conn.Flush(); err != nil {
		return nil, err
	}

	// Construct chosen labels
	out := make([]ot.Label, N)
	for j := 0; j < N; j++ {
		var b ot.LabelData

		for bit := 0; bit < IKNPK; bit++ {
			byteRow := j / 8
			bitPos := uint(j % 8)
			var rowBit byte
			if flags[j] {
				rowBit = (T1[bit][byteRow] >> bitPos) & 1
			} else {
				rowBit = (T0[bit][byteRow] >> bitPos) & 1
			}
			bytePos := bit / 8
			inner := uint(7 - (bit % 8))
			if rowBit == 1 {
				b[bytePos] |= (1 << inner)
			}
		}

		out[j].SetData(&b)
	}

	return out, nil
}
