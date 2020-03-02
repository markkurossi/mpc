//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
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

func Garbler(conn *Conn, circ *Circuit, inputs []*big.Int, key []byte,
	verbose bool) ([]*big.Int, error) {

	timing := NewTiming()

	garbled, err := circ.Garble(key)
	if err != nil {
		return nil, err
	}

	timing.Sample("Garble", nil)

	// Send garbled tables.
	if err := conn.SendUint32(len(garbled.Gates)); err != nil {
		return nil, err
	}
	for _, data := range garbled.Gates {
		if err := conn.SendUint32(len(data)); err != nil {
			return nil, err
		}
		for _, d := range data {
			if err := conn.SendData(d); err != nil {
				return nil, err
			}
		}
	}

	// Select our inputs.
	var n1 [][]byte
	var w int
	for idx, io := range circ.N1 {
		var input *big.Int
		if idx < len(inputs) {
			input = inputs[idx]
		}
		for i := 0; i < io.Size; i++ {
			wire := garbled.Wires[w]
			w++

			var n []byte

			if input != nil && input.Bit(i) == 1 {
				n = wire.Label1.Bytes()
			} else {
				n = wire.Label0.Bytes()
			}
			n1 = append(n1, n)
		}
	}

	// Send our inputs.
	for idx, i := range n1 {
		if verbose && false {
			fmt.Printf("N1[%d]:\t%x\n", idx, i)
		}
		if err := conn.SendData(i); err != nil {
			return nil, err
		}
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048, garbled.Wires)
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

	ioStats := conn.Stats
	timing.Sample("Xfer", []string{FileSize(ioStats.Sum()).String()})

	// Process messages.

	var xfer *ot.SenderXfer
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

			xfer, err = sender.NewTransfer(bit)
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
			for i := 0; i < circ.N3.Size(); i++ {
				label, err := conn.ReceiveData()
				if err != nil {
					return nil, err
				}
				wire := garbled.Wires[circ.NumWires-circ.N3.Size()+i]

				var bit uint
				if bytes.Compare(label, wire.Label0.Bytes()) == 0 {
					bit = 0
				} else if bytes.Compare(label, wire.Label1.Bytes()) == 0 {
					bit = 1
				} else {
					return nil, fmt.Errorf("Unknown label %x for result %d",
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

	return circ.N3.Split(result), nil
}
