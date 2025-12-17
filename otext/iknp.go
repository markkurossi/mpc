package otext

import (
	"errors"
	"io"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

const (
	// IKNPK defines the number of base-OTs.
	IKNPK      = 128
	labelBytes = 16
)

// The roles.
const (
	SenderRole   = 0
	ReceiverRole = 1
)

// IKNPExt implements the IKNP OT extension.
type IKNPExt struct {
	base    ot.OT
	conn    *p2p.Conn
	role    int
	k       int
	seedS   [][]byte
	seed0   [][]byte
	seed1   [][]byte
	choices []bool // store base-OT choice bits for sender
}

// NewIKNPExt creates a new IKNP OT extension.
func NewIKNPExt(base ot.OT, conn *p2p.Conn, role int) *IKNPExt {
	return &IKNPExt{
		base: base,
		conn: conn,
		role: role,
		k:    IKNPK,
	}
}

func packLabel(b []byte) ot.Label {
	var d ot.LabelData
	copy(d[:], b[:labelBytes])
	var l ot.Label
	l.SetData(&d)
	return l
}

// Setup phase runs k=128 base OTs.
func (e *IKNPExt) Setup(r io.Reader) error {
	if e.role == SenderRole {
		// base OT receiver: choose random choice bits and call base.Receive.
		choices := make([]bool, e.k)
		var buf [IKNPK / 8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return err
		}
		for i := 0; i < e.k; i++ {
			choices[i] = ((buf[i/8] >> uint(i%8)) & 1) == 1
		}

		labels := make([]ot.Label, e.k)
		if err := e.base.Receive(choices, labels); err != nil {
			return err
		}

		e.seedS = make([][]byte, e.k)
		for i := 0; i < e.k; i++ {
			var d ot.LabelData
			labels[i].GetData(&d)
			b := make([]byte, 16)
			copy(b, d[:])
			e.seedS[i] = b
		}
		// store choice bits for later expansion
		e.choices = choices

		return nil

	} else {
		// base OT sender: prepare k label pairs and call base.Send
		e.seed0 = make([][]byte, e.k)
		e.seed1 = make([][]byte, e.k)

		wires := make([]ot.Wire, e.k)
		for i := 0; i < e.k; i++ {
			l0, err := ot.NewLabel(r)
			if err != nil {
				return err
			}
			l1, err := ot.NewLabel(r)
			if err != nil {
				return err
			}

			var d0, d1 ot.LabelData
			l0.GetData(&d0)
			l1.GetData(&d1)

			b0 := make([]byte, 16)
			b1 := make([]byte, 16)
			copy(b0, d0[:])
			copy(b1, d1[:])
			e.seed0[i] = b0
			e.seed1[i] = b1

			wires[i] = ot.Wire{L0: l0, L1: l1}
		}
		return e.base.Send(wires)
	}
}

// ExpandSend implements the sender side of IKNP.
func (e *IKNPExt) ExpandSend(n int) ([]ot.Wire, error) {
	if e.role != SenderRole {
		return nil, errors.New("wrong role")
	}

	if e.seedS == nil || len(e.seedS) != e.k {
		return nil, errors.New("seedS not initialized; call Setup() first")
	}
	if e.choices == nil || len(e.choices) != e.k {
		return nil, errors.New("choices not initialized; call Setup() first")
	}
	if n <= 0 {
		return nil, errors.New("n must be positive")
	}

	rowBytes := (n + 7) / 8
	total := e.k * rowBytes

	// receive U (k xor-rows) from receiver: U = T0 ^ T1 for each row
	U, err := e.conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	if len(U) < total {
		return nil, errors.New("not enough U")
	}

	// generate T0 rows from seedS and, if choice bit == 1, PRG(seedS)
	// ^ U_row would be T0?  The standard IKNP computation yields rows
	// equal to T0 for all i after this step.  We'll compute rows =
	// PRG(seedS) XOR (choices[i] ? U_row : 0) which yields T0 in
	// either case.
	rows := make([][]byte, e.k)
	for i := 0; i < e.k; i++ {
		rows[i] = make([]byte, rowBytes)
		prgAESCTR(e.seedS[i], rows[i])

		if e.choices[i] {
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
		var b0 [16]byte
		var b1 [16]byte

		for bit := 0; bit < 128; bit++ {
			if bit >= e.k {
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

		L0 := packLabel(b0[:])
		L1 := packLabel(b1[:])

		wires[j] = ot.Wire{L0: L0, L1: L1}
	}

	return wires, nil
}

// ExpandReceive implemenets the receiver side of IKNP.
func (e *IKNPExt) ExpandReceive(flags []bool) ([]ot.Label, error) {
	if e.role != ReceiverRole {
		return nil, errors.New("wrong role")
	}

	if e.seed0 == nil || len(e.seed0) != e.k || e.seed1 == nil || len(e.seed1) != e.k {
		return nil, errors.New("seed0/seed1 not initialized; call Setup() first")
	}
	N := len(flags)
	if N == 0 {
		return nil, errors.New("flags empty")
	}
	rowBytes := (N + 7) / 8

	T0 := make([][]byte, e.k)
	T1 := make([][]byte, e.k)
	for i := 0; i < e.k; i++ {
		T0[i] = make([]byte, rowBytes)
		T1[i] = make([]byte, rowBytes)
		prgAESCTR(e.seed0[i], T0[i])
		prgAESCTR(e.seed1[i], T1[i])
	}

	// send U rows (T0 xor T1) concatenated
	U := make([]byte, e.k*rowBytes)
	for i := 0; i < e.k; i++ {
		for j := 0; j < rowBytes; j++ {
			U[i*rowBytes+j] = T0[i][j] ^ T1[i][j]
		}
	}
	if err := e.conn.SendData(U); err != nil {
		return nil, err
	}
	if err := e.conn.Flush(); err != nil {
		return nil, err
	}

	// Construct chosen labels
	out := make([]ot.Label, N)
	for j := 0; j < N; j++ {
		var b [16]byte
		for bit := 0; bit < 128; bit++ {
			if bit >= e.k {
				break
			}
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
		out[j] = packLabel(b[:])
	}
	return out, nil
}
