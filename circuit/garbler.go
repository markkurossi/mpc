//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
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

const (
	OP_OT = iota
	OP_RESULT
	OP_CIRCUIT
)

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

func Garbler(conn *p2p.Conn, circ *Circuit, inputs *big.Int, verbose bool) (
	[]*big.Int, error) {

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
	for _, data := range garbled.Gates {
		if err := conn.SendUint32(len(data)); err != nil {
			return nil, err
		}
		for _, d := range data {
			if err := conn.SendLabel(d); err != nil {
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
		if err := conn.SendLabel(i); err != nil {
			return nil, err
		}
	}
	ioStats := conn.Stats
	timing.Sample("Xfer", []string{FileSize(ioStats.Sum()).String()})
	if verbose {
		fmt.Printf(" - Processing messages...\n")
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048)
	if err != nil {
		return nil, err
	}

	// Send our public key.
	pub := sender.PublicKey()
	data := pub.N.Bytes()
	if err := conn.SendData(data); err != nil {
		return nil, err
	}
	if err := conn.SendUint32(pub.E); err != nil {
		return nil, err
	}
	conn.Flush()

	ioStats = conn.Stats.Sub(ioStats)
	timing.Sample("OT Init", []string{FileSize(ioStats.Sum()).String()})

	// Init wires the peer is allowed to OT.
	allowedOTs := make(map[int]bool)
	for bit := 0; bit < circ.Inputs[1].Size; bit++ {
		allowedOTs[circ.Inputs[0].Size+bit] = true
	}

	// Process messages.

	lastOT := time.Now()
	done := false
	result := big.NewInt(0)

	for !done {
		op, err := conn.ReceiveUint32()
		if err != nil {
			return nil, err
		}

		switch op {
		case OP_OT:
			bit, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			if !allowedOTs[bit] {
				return nil, fmt.Errorf("peer can't OT wire %d", bit)
			}
			allowedOTs[bit] = false

			wire := garbled.Wires[bit]

			m0Data := wire.L0.Bytes()
			m1Data := wire.L1.Bytes()

			xfer, err := sender.NewTransfer(m0Data, m1Data)
			if err != nil {
				return nil, err
			}

			x0, x1 := xfer.RandomMessages()
			if err := conn.SendData(x0); err != nil {
				return nil, err
			}
			if err := conn.SendData(x1); err != nil {
				return nil, err
			}
			conn.Flush()

			v, err := conn.ReceiveData()
			if err != nil {
				return nil, err
			}
			xfer.ReceiveV(v)

			m0p, m1p, err := xfer.Messages()
			if err != nil {
				return nil, err
			}
			if err := conn.SendData(m0p); err != nil {
				return nil, err
			}
			if err := conn.SendData(m1p); err != nil {
				return nil, err
			}
			conn.Flush()
			lastOT = time.Now()

		case OP_RESULT:
			for i := 0; i < circ.Outputs.Size(); i++ {
				label, err := conn.ReceiveLabel()
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
					return nil, fmt.Errorf("Unknown label %s for result %d",
						label, i)
				}
				result = big.NewInt(0).SetBit(result, i, bit)
			}
			data := result.Bytes()
			if err := conn.SendData(data); err != nil {
				return nil, err
			}
			conn.Flush()
			done = true
		}
	}
	ioStats = conn.Stats.Sub(ioStats)
	timing.Sample("Eval", []string{FileSize(ioStats.Sum()).String()}).
		SubSample("OT", lastOT)
	if verbose {
		timing.Print()
	}

	return circ.Outputs.Split(result), nil
}
