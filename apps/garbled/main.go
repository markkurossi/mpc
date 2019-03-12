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
	"time"

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

type FileSize uint64

func (s FileSize) String() string {
	if s > 1024*1024*1024*1024 {
		return fmt.Sprintf("%dGB", s/1024*1024*1024)
	} else if s > 1024*1024*1024 {
		return fmt.Sprintf("%dMB", s/1024*1024)
	} else if s > 1024*1024 {
		return fmt.Sprintf("%dkB", s/1024)
	} else {
		return fmt.Sprintf("%dB", s)
	}
}

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
		err = garblerMode(circ, big.NewInt(int64(*input)))
	} else {
		err = evaluatorMode(circ, big.NewInt(int64(*input)))
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

func garblerMode(circ *circuit.Circuit, input *big.Int) error {
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

func serveConnection(conn net.Conn, circ *circuit.Circuit,
	input *big.Int) error {
	fmt.Printf("Serving connetion from %s\n", conn.RemoteAddr())
	defer conn.Close()

	// Assign labels to wires.
	start := time.Now()
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
	t := time.Now()
	fmt.Printf("Labels:\t%s\n", t.Sub(start))
	start = t

	garbled := make(map[int][][]byte)

	for id, gate := range circ.Gates {
		data, err := gate.Garble(wires, enc)
		if err != nil {
			return err
		}
		garbled[id] = data
	}
	t = time.Now()
	fmt.Printf("Garble:\t%s\n", t.Sub(start))
	start = t

	// Send garbled tables.
	var size FileSize
	for id, data := range garbled {
		if err := sendUint32(conn, id); err != nil {
			return err
		}
		size += 4
		if err := sendUint32(conn, len(data)); err != nil {
			return err
		}
		size += 4
		for _, d := range data {
			if err := sendData(conn, d); err != nil {
				return err
			}
			size += FileSize(4 + len(d))
		}
	}

	// Select our inputs.
	var n1 [][]byte
	for i := 0; i < circ.N1; i++ {
		wire := wires[i]

		var n []byte

		if input.Bit(i) == 1 {
			n = wire.Label1
		} else {
			n = wire.Label0
		}
		n1 = append(n1, n)
	}

	// Send our inputs.
	for idx, i := range n1 {
		if verbose {
			fmt.Printf("N1[%d]:\t%x\n", idx, i)
		}
		if err := sendData(conn, i); err != nil {
			return err
		}
		size += FileSize(4 + len(i))
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048, wires)
	if err != nil {
		return err
	}

	// Send our public key.
	pub := sender.PublicKey()
	data := pub.N.Bytes()
	if err := sendData(conn, data); err != nil {
		return err
	}
	size += FileSize(4 + len(data))
	if err := sendUint32(conn, pub.E); err != nil {
		return err
	}
	size += 4
	t = time.Now()
	fmt.Printf("Xfer:\t%s\t%s\n", t.Sub(start), size)
	start = t

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
			result := big.NewInt(0)

			for i := 0; i < circ.N3; i++ {
				label, err := receiveData(conn)
				if err != nil {
					return err
				}
				wire := wires[circ.NumWires-circ.N3+i]

				var bit uint
				if bytes.Compare(label, wire.Label0) == 0 {
					bit = 0
				} else if bytes.Compare(label, wire.Label1) == 0 {
					bit = 1
				} else {
					return fmt.Errorf("Unknown label %x for result %d",
						label, i)
				}
				result = big.NewInt(0).SetBit(result, i, bit)
			}
			if err := sendData(conn, result.Bytes()); err != nil {
				return err
			}
			fmt.Printf("Result: %v\n", result)
			done = true
		}
	}
	t = time.Now()
	fmt.Printf("Eval:\t%s\n", t.Sub(start))
	start = t

	return nil
}

func evaluatorMode(circ *circuit.Circuit, input *big.Int) error {
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
		if input.Bit(i) == 1 {
			bit = 1
		} else {
			bit = 0
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

	fmt.Printf("Result: %v\n", val)
	fmt.Printf("Result: %x\n", val.Bytes())

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

func result(conn net.Conn, labels [][]byte) (*big.Int, error) {
	if err := sendUint32(conn, OP_RESULT); err != nil {
		return nil, err
	}
	for _, l := range labels {
		if err := sendData(conn, l); err != nil {
			return nil, err
		}
	}

	result, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(result), nil
}
