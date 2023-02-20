//
// garbler.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// Protocol operation codes.
const (
	OpOT = iota
	OpResult
	OpCircuit
	OpReturn
)

// FileSize specifies a file (or data transfer) size in bytes.
type FileSize uint64

func (s FileSize) String() string {
	if s > 1000*1000*1000*1000 {
		return fmt.Sprintf("%dTB", s/(1000*1000*1000*1000))
	} else if s > 1000*1000*1000 {
		return fmt.Sprintf("%dGB", s/(1000*1000*1000))
	} else if s > 1000*1000 {
		return fmt.Sprintf("%dMB", s/(1000*1000))
	} else if s > 1000 {
		return fmt.Sprintf("%dkB", s/1000)
	} else {
		return fmt.Sprintf("%dB", s)
	}
}

// Garbler runs the garbler on the P2P network.
func Garbler(conn *p2p.Conn, oti ot.OT, circ *Circuit, inputs *big.Int,
	verbose bool) ([]*big.Int, error) {

	timing := NewTiming()
	if verbose {
		fmt.Printf(" - Garbling...\n")
	}

	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
	}

	garbled, err := circ.Garble(key[:])
	if err != nil {
		return nil, err
	}

	timing.Sample("Garble", nil)

	// Send program info.
	if verbose {
		fmt.Printf(" - Sending garbled circuit...\n")
	}
	if err := conn.SendData(key[:]); err != nil {
		return nil, err
	}

	// Send garbled tables.
	if err := conn.SendUint32(len(garbled.Gates)); err != nil {
		return nil, err
	}
	var labelData ot.LabelData
	for _, data := range garbled.Gates {
		if err := conn.SendUint32(len(data)); err != nil {
			return nil, err
		}
		for _, d := range data {
			if err := conn.SendLabel(d, &labelData); err != nil {
				return nil, err
			}
		}
	}

	// Select our inputs.
	var n1 []ot.Label
	for i := 0; i < circ.Inputs[0].Size; i++ {
		wire := garbled.Wires[i]

		var n ot.Label

		if inputs.Bit(i) == 1 {
			n = wire.L1
		} else {
			n = wire.L0
		}
		n1 = append(n1, n)
	}

	// Send our inputs.
	for idx, i := range n1 {
		if verbose && false {
			fmt.Printf("N1[%d]:\t%s\n", idx, i)
		}
		if err := conn.SendLabel(i, &labelData); err != nil {
			return nil, err
		}
	}
	ioStats := conn.Stats
	timing.Sample("Xfer", []string{FileSize(ioStats.Sum()).String()})
	if verbose {
		fmt.Printf(" - Processing messages...\n")
	}

	// Init oblivious transfer.
	err = oti.InitSender(conn)
	if err != nil {
		return nil, err
	}
	xfer := conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("OT Init", []string{FileSize(xfer.Sum()).String()})

	// Init wires the peer is allowed to OT.
	allowedOTs := make(map[int]bool)
	for bit := 0; bit < circ.Inputs[1].Size; bit++ {
		allowedOTs[circ.Inputs[0].Size+bit] = true
	}

	// Process messages.

	lastOT := time.Now()
	done := false
	result := big.NewInt(0)

	var label ot.Label

	for !done {
		op, err := conn.ReceiveUint32()
		if err != nil {
			return nil, err
		}

		switch op {
		case OpOT:
			offset, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			count, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			for i := 0; i < count; i++ {
				if !allowedOTs[offset+i] {
					return nil, fmt.Errorf("peer can't OT wire %d", offset+i)
				}
				allowedOTs[offset+i] = false
			}
			err = oti.Send(garbled.Wires[offset : offset+count])
			if err != nil {
				return nil, err
			}
			lastOT = time.Now()

		case OpResult:
			for i := 0; i < circ.Outputs.Size(); i++ {
				err := conn.ReceiveLabel(&label, &labelData)
				if err != nil {
					return nil, err
				}
				wire := garbled.Wires[circ.NumWires-circ.Outputs.Size()+i]

				var bit uint
				if label.Equal(wire.L0) {
					bit = 0
				} else if label.Equal(wire.L1) {
					bit = 1
				} else {
					return nil, fmt.Errorf("unknown label %s for result %d",
						label, i)
				}
				result = big.NewInt(0).SetBit(result, i, bit)
			}
			data := result.Bytes()
			if err := conn.SendData(data); err != nil {
				return nil, err
			}
			if err := conn.Flush(); err != nil {
				return nil, err
			}
			done = true
		}
	}
	xfer = conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Eval", []string{FileSize(xfer.Sum()).String()}).
		SubSample("OT", lastOT)
	if verbose {
		timing.Print(conn.Stats.Sent, conn.Stats.Recvd)
	}

	return circ.Outputs.Split(result), nil
}
