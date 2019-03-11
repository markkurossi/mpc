//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/ot"
)

const (
	k = 128

	OP_OT = iota
	OP_RESULT
)

var (
	port          = ":8080"
	DecryptFailed = errors.New("Decrypt failed")
	verbose       = false
)

func main() {
	garbler := flag.Bool("g", false, "Garbler / Evaluator mode")
	file := flag.String("c", "", "Circuit file")
	input := flag.Int("i", 0, "Circuit input")
	fVerbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	verbose = *fVerbose

	if len(*file) == 0 {
		fmt.Printf("Circuit file not specified\n")
		return
	}

	circ, err := loadCircuit(*file)
	if err != nil {
		fmt.Printf("Failed to parse circuit file '%s': %s\n", *file, err)
		os.Exit(1)
	}
	fmt.Printf("Circuit: %v\n", circ)
	fmt.Printf("Input: %d\n", *input)

	if *garbler {
		err = garblerMode(circ, *input)
	} else {
		err = evaluatorMode(circ, *input)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func loadCircuit(file string) (*circuit.Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return circuit.Parse(f)
}

func garblerMode(circ *circuit.Circuit, input int) error {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	fmt.Printf("Listening for connections at %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		err = serveConnection(conn, circ, input)
		if err != nil {
			return err
		}
	}
	return nil
}

func enc(a, b, c []byte) []byte {
	var key [32]byte

	copy(key[0:], a)
	copy(key[16:], b)

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		panic(err)
	}

	block := cipher.BlockSize()

	result := make([]byte, 2*block)
	cipher.Encrypt(result[0:block], result[0:block])
	cipher.Encrypt(result[block:], c)

	return result
}

func serveConnection(conn net.Conn, circ *circuit.Circuit, input int) error {
	fmt.Printf("Serving connetion from %s\n", conn.RemoteAddr())
	defer conn.Close()

	// Assign labels to wires.
	wires := make(ot.Inputs)
	for w := 0; w < circ.NumWires; w++ {
		l0 := make([]byte, k/8)
		if _, err := rand.Read(l0); err != nil {
			return err
		}

		l1 := make([]byte, k/8)
		if _, err := rand.Read(l1); err != nil {
			return err
		}
		wires[w] = ot.Wire{
			Label0: l0,
			Label1: l1,
		}
	}

	garbled := make(map[int][][]byte)

	for id, gate := range circ.Gates {
		data, err := gate.Garble(wires, enc)
		if err != nil {
			return err
		}
		garbled[id] = data
	}

	// Send garbled tables.
	for id, data := range garbled {
		if err := sendUint32(conn, id); err != nil {
			return err
		}
		if err := sendUint32(conn, len(data)); err != nil {
			return err
		}
		for _, d := range data {
			if err := sendData(conn, d); err != nil {
				return err
			}
		}
	}

	// Select our inputs.
	var n1 [][]byte
	for i := 0; i < circ.N1; i++ {
		wire := wires[i]

		var n []byte

		if (input & (1 << uint(i))) == 0 {
			n = wire.Label0
		} else {
			n = wire.Label1
		}
		n1 = append(n1, n)
	}

	// Send our inputs.
	for idx, i := range n1 {
		fmt.Printf("N1[%d]:\t%x\n", idx, i)
		if err := sendData(conn, i); err != nil {
			return err
		}
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048, wires)
	if err != nil {
		return err
	}

	// Send our public key.
	pub := sender.PublicKey()
	if err := sendData(conn, pub.N.Bytes()); err != nil {
		return err
	}
	if err := sendUint32(conn, pub.E); err != nil {
		return err
	}

	// Process messages.
	var xfer *ot.SenderXfer
	done := false
	for !done {
		op, err := receiveUint32(conn)
		if err != nil {
			return err
		}
		switch op {
		case OP_OT:
			bit, err := receiveUint32(conn)
			if err != nil {
				return err
			}
			xfer, err = sender.NewTransfer(bit)
			if err != nil {
				return err
			}

			x0, x1 := xfer.RandomMessages()
			if err := sendData(conn, x0); err != nil {
				return err
			}
			if err := sendData(conn, x1); err != nil {
				return err
			}

			v, err := receiveData(conn)
			if err != nil {
				return err
			}
			xfer.ReceiveV(v)

			m0p, m1p, err := xfer.Messages()
			if err != nil {
				return err
			}
			if err := sendData(conn, m0p); err != nil {
				return err
			}
			if err := sendData(conn, m1p); err != nil {
				return err
			}

		case OP_RESULT:
			var result int

			for i := 0; i < circ.N3; i++ {
				label, err := receiveData(conn)
				if err != nil {
					return err
				}
				wire := wires[circ.NumWires-circ.N3+i]

				var bit int
				if bytes.Compare(label, wire.Label0) == 0 {
					bit = 0
				} else if bytes.Compare(label, wire.Label1) == 0 {
					bit = 1
				} else {
					return fmt.Errorf("Unknown lable %x for result %d",
						label, i)
				}
				result |= (bit << uint(i))
			}
			if err := sendUint32(conn, result); err != nil {
				return err
			}
			fmt.Printf("Result: %d\n", result)
			done = true
		}
	}

	return nil
}

func evaluatorMode(circ *circuit.Circuit, input int) error {
	conn, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	defer conn.Close()

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
			if verbose {
				fmt.Printf("G%d.%d\t%x\n", i, j, v)
			}
			values = append(values, v)
		}
		garbled[id] = values
	}

	wires := make(map[int][]byte)

	// Receive peer inputs.
	for i := 0; i < circ.N1; i++ {
		n, err := receiveData(conn)
		if err != nil {
			return err
		}
		fmt.Printf("N1[%d]:\t%x\n", i, n)
		wires[i] = n
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
		if (input & (1 << uint(i))) == 0 {
			bit = 0
		} else {
			bit = 1
		}

		n, err := receive(conn, receiver, circ.N1+i, bit)
		if err != nil {
			return err
		}
		fmt.Printf("N2[%d]:\t%x\n", i, n)
		wires[circ.N1+i] = n
	}

	// Evaluate gates.
	for id := 0; id < circ.NumGates; id++ {
		gate := circ.Gates[id]
		if verbose {
			fmt.Printf("Evaluating gate %d %s\n", id, gate.Op)
		}

		output, err := gate.Eval(wires, dec, garbled[id])
		if err != nil {
			return err
		}
		wires[gate.Outputs[0]] = output
	}

	var labels [][]byte

	for i := 0; i < circ.N3; i++ {
		r := wires[circ.NumWires-circ.N3+i]
		labels = append(labels, r)
	}

	val, err := result(conn, labels)
	if err != nil {
		return err
	}

	fmt.Printf("Result: %d\n", val)

	return nil
}

func dec(a, b, data []byte) ([]byte, error) {
	var key [32]byte

	copy(key[0:], a)
	copy(key[16:], b)

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	block := cipher.BlockSize()
	result := make([]byte, block)
	cipher.Decrypt(result, data[:block])

	for _, b := range result {
		if b != 0 {
			return nil, DecryptFailed
		}
	}

	cipher.Decrypt(result, data[block:])
	return result, nil
}

func receive(conn net.Conn, receiver *ot.Receiver, wire, bit int) (
	[]byte, error) {

	if err := sendUint32(conn, OP_OT); err != nil {
		return nil, err
	}
	if err := sendUint32(conn, wire); err != nil {
		return nil, err
	}

	xfer, err := receiver.NewTransfer(bit)
	if err != nil {
		return nil, err
	}

	x0, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	x1, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	err = xfer.ReceiveRandomMessages(x0, x1)
	if err != nil {
		return nil, err
	}

	v := xfer.V()
	if err := sendData(conn, v); err != nil {
		return nil, err
	}

	m0p, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	m1p, err := receiveData(conn)
	if err != nil {
		return nil, err
	}

	err = xfer.ReceiveMessages(m0p, m1p, nil)
	if err != nil {
		return nil, err
	}

	m, _ := xfer.Message()
	return m, nil
}

func result(conn net.Conn, labels [][]byte) (int, error) {
	if err := sendUint32(conn, OP_RESULT); err != nil {
		return 0, err
	}
	for _, l := range labels {
		if err := sendData(conn, l); err != nil {
			return 0, err
		}
	}

	result, err := receiveUint32(conn)
	if err != nil {
		return 0, err
	}
	return result, nil
}
