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
	"github.com/markkurossi/mpc/p2p"
)

var (
	debug = false
)

func Evaluator(conn *p2p.Conn, circ *Circuit, inputs []*big.Int, verbose bool) (
	[]*big.Int, error) {

	timing := NewTiming()

	garbled := make([][][]byte, circ.NumGates)

	// Receive program info.
	if verbose {
		fmt.Printf(" - Waiting for circuit info...\n")
	}
	key, err := conn.ReceiveData()
	if err != nil {
		return nil, err
	}

	// Receive garbled tables.
	timing.Sample("Wait", nil)
	if verbose {
		fmt.Printf(" - Receiving garbled circuit...\n")
	}
	count, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	if count != circ.NumGates {
		return nil, fmt.Errorf("wrong number of gates: got %d, expected %d",
			count, circ.NumGates)
	}
	for i := 0; i < circ.NumGates; i++ {
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
		garbled[i] = values
	}

	wires := make([]*ot.Label, circ.NumWires)

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
	if verbose {
		fmt.Printf(" - Querying our inputs...\n")
	}
	var w int
	for idx, io := range circ.N2 {
		var input *big.Int
		if idx < len(inputs) {
			input = inputs[idx]
		}
		for i := 0; i < io.Size; i++ {
			if err := conn.SendUint32(OP_OT); err != nil {
				return nil, err
			}
			n, err := conn.Receive(receiver, uint(circ.N1.Size()+w),
				input.Bit(i))
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
	if verbose {
		fmt.Printf(" - Evaluating circuit...\n")
	}
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

	// Resolve result values.
	if err := conn.SendUint32(OP_RESULT); err != nil {
		return nil, err
	}
	for _, l := range labels {
		if err := conn.SendData(l.Bytes()); err != nil {
			return nil, err
		}
	}
	conn.Flush()

	result, err := conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	raw := big.NewInt(0).SetBytes(result)

	xfer = conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Result", []string{FileSize(xfer.Sum()).String()})
	if verbose {
		timing.Print()
	}

	return circ.N3.Split(raw), nil
}
