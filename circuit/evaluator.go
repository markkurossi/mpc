//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/rsa"
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/ot"
)

var (
	debug = false
)

func Evaluator(conn *Conn, circ *Circuit, inputs []*big.Int, key []byte,
	verbose bool) ([]*big.Int, error) {

	timing := NewTiming()

	garbled := make(map[int][][]byte)

	// Receive garbled tables.
	for i := 0; i < circ.NumGates; i++ {
		id, err := conn.ReceiveUint32()
		if err != nil {
			return nil, err
		}
		count, err := conn.ReceiveUint32()
		if err != nil {
			return nil, err
		}

		var values [][]byte
		for j := 0; j < count; j++ {
			v, err := conn.ReceiveData()
			if err != nil {
				return nil, err
			}
			if debug {
				fmt.Printf("G%d.%d\t%x\n", i, j, v)
			}
			values = append(values, v)
		}
		garbled[id] = values
	}

	wires := make(map[Wire]*ot.Label)

	// Receive peer inputs.
	for i := 0; i < circ.N1.Size(); i++ {
		n, err := conn.ReceiveData()
		if err != nil {
			return nil, err
		}
		if debug {
			fmt.Printf("N1[%d]:\t%x\n", i, n)
		}
		wires[Wire(i)] = ot.LabelFromData(n)
	}

	// Init oblivious transfer.
	pubN, err := conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	pubE, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	pub := &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(pubN),
		E: pubE,
	}
	receiver, err := ot.NewReceiver(pub)
	if err != nil {
		return nil, err
	}
	ioStats := conn.Stats
	timing.Sample("Recv", []string{FileSize(ioStats.Sum()).String()})

	// Query our inputs.
	var w int
	for idx, io := range circ.N2 {
		var input *big.Int
		if idx < len(inputs) {
			input = inputs[idx]
		}
		for i := 0; i < io.Size; i++ {
			var bit int
			if input.Bit(i) == 1 {
				bit = 1
			} else {
				bit = 0
			}

			n, err := conn.Receive(receiver, circ.N1.Size()+w, bit)
			if err != nil {
				return nil, err
			}
			if debug {
				fmt.Printf("N2[%d]:\t%x\n", w, n)
			}
			wires[Wire(circ.N1.Size()+w)] = ot.LabelFromData(n)
			w++
		}
	}
	xfer := conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Inputs", []string{FileSize(xfer.Sum()).String()})

	// Evaluate gates.
	err = circ.Eval(key[:], wires, garbled)
	if err != nil {
		return nil, err
	}
	timing.Sample("Eval", nil)

	var labels []*ot.Label

	for i := 0; i < circ.N3.Size(); i++ {
		r := wires[Wire(circ.NumWires-circ.N3.Size()+i)]
		labels = append(labels, r)
	}

	raw, err := conn.Result(labels)
	if err != nil {
		return nil, err
	}
	xfer = conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Result", []string{FileSize(xfer.Sum()).String()})
	if verbose {
		timing.Print()
	}

	return circ.N3.Split(raw), nil
}
