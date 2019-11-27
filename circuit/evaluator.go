//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bufio"
	"crypto/rsa"
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/ot"
)

var (
	debug = false
)

func Evaluator(conn *bufio.ReadWriter, circ *Circuit, input *big.Int,
	key []byte) error {

	garbled := make(map[int][][]byte)

	// Receive garbled tables.
	for i := 0; i < circ.NumGates; i++ {
		id, err := receiveUint32(conn)
		if err != nil {
			return err
		}
		count, err := receiveUint32(conn)
		if err != nil {
			return err
		}

		var values [][]byte
		for j := 0; j < count; j++ {
			v, err := receiveData(conn)
			if err != nil {
				return err
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
	for i := 0; i < circ.N1; i++ {
		n, err := receiveData(conn)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Printf("N1[%d]:\t%x\n", i, n)
		}
		wires[Wire(i)] = ot.LabelFromData(n)
	}

	// Init oblivious transfer.
	pubN, err := receiveData(conn)
	if err != nil {
		return err
	}
	pubE, err := receiveUint32(conn)
	if err != nil {
		return err
	}
	pub := &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(pubN),
		E: pubE,
	}
	receiver, err := ot.NewReceiver(pub)
	if err != nil {
		return err
	}

	// Query our inputs.
	for i := 0; i < circ.N2; i++ {
		var bit int
		if input.Bit(i) == 1 {
			bit = 1
		} else {
			bit = 0
		}

		n, err := receive(conn, receiver, circ.N1+i, bit)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Printf("N2[%d]:\t%x\n", i, n)
		}
		wires[Wire(circ.N1+i)] = ot.LabelFromData(n)
	}

	// Evaluate gates.
	err = circ.Eval(key[:], wires, garbled)
	if err != nil {
		return err
	}

	var labels []*ot.Label

	for i := 0; i < circ.N3; i++ {
		r := wires[Wire(circ.NumWires-circ.N3+i)]
		labels = append(labels, r)
	}

	val, err := result(conn, labels)
	if err != nil {
		return err
	}

	fmt.Printf("Result: %v\n", val)
	fmt.Printf("Result: 0b%s\n", val.Text(2))
	fmt.Printf("Result: 0x%x\n", val.Bytes())

	return nil
}
