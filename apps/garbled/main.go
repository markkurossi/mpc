//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
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
	debug         = false
	key           [32]byte // XXX
)

type FileSize uint64

func (s FileSize) String() string {
	if s > 1000*1000*1000*1000 {
		return fmt.Sprintf("%d TB", s/(1000*1000*1000*1000))
	} else if s > 1000*1000*1000 {
		return fmt.Sprintf("%d GB", s/(1000*1000*1000))
	} else if s > 1000*1000 {
		return fmt.Sprintf("%d MB", s/(1000*1000))
	} else if s > 1000 {
		return fmt.Sprintf("%d kB", s/1000)
	} else {
		return fmt.Sprintf("%d B", s)
	}
}

func main() {
	garbler := flag.Bool("g", false, "Garbler / Evaluator mode")
	compile := flag.Bool("c", false, "Compile MPCL to circuit")
	input := flag.Uint64("i", 0, "Circuit input")
	fVerbose := flag.Bool("v", false, "Verbose output")
	fDebug := flag.Bool("d", false, "Debug output")
	flag.Parse()

	verbose = *fVerbose
	debug = *fDebug

	var circ *circuit.Circuit
	var err error

	if len(flag.Args()) == 0 {
		fmt.Printf("No input files\n")
		os.Exit(1)
	}

	for _, arg := range flag.Args() {
		if strings.HasSuffix(arg, ".circ") {
			circ, err = loadCircuit(arg)
			if err != nil {
				fmt.Printf("Failed to parse circuit file '%s': %s\n", arg, err)
				os.Exit(1)
			}
		} else if strings.HasSuffix(arg, ".mpcl") {
			circ, err = compileCircuit(arg)
			if err != nil {
				fmt.Printf("Failed to compile input file '%s': %s\n", arg, err)
				os.Exit(1)
			}
			if *compile {
				out := arg[0:len(arg)-4] + "circ"
				f, err := os.Create(out)
				if err != nil {
					fmt.Printf("Failed to create output file '%s': %s\n",
						out, err)
					os.Exit(1)
				}
				circ.Marshal(f)
				f.Close()
				return
			}
		} else {
			fmt.Printf("Unknown file type '%s'\n", arg)
			os.Exit(1)
		}
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

func compileCircuit(file string) (*circuit.Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	parser := compiler.NewParser(file, f)
	unit, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return unit.Compile()
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
		fmt.Printf("New connection from %s\n", conn.RemoteAddr())

		io := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		err = serveConnection(io, circ, input)

		conn.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func serveConnection(conn *bufio.ReadWriter, circ *circuit.Circuit,
	input *big.Int) error {

	start := time.Now()

	garbled, err := circ.Garble(key[:])
	if err != nil {
		return err
	}

	t := time.Now()
	fmt.Printf("Garble:\t%s\n", t.Sub(start))
	start = t

	// Send garbled tables.
	var size FileSize
	for id, data := range garbled.Gates {
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
		wire := garbled.Wires[i]

		var n []byte

		if input.Bit(i) == 1 {
			n = wire.Label1.Bytes()
		} else {
			n = wire.Label0.Bytes()
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
	sender, err := ot.NewSender(2048, garbled.Wires)
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
	conn.Flush()
	t = time.Now()
	fmt.Printf("Xfer:\t%s\t%s\n", t.Sub(start), size)
	start = t

	// Process messages.
	var xfer *ot.SenderXfer
	lastOT := start
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
			conn.Flush()

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
			conn.Flush()
			lastOT = time.Now()

		case OP_RESULT:
			result := big.NewInt(0)

			for i := 0; i < circ.N3; i++ {
				label, err := receiveData(conn)
				if err != nil {
					return err
				}
				wire := garbled.Wires[circ.NumWires-circ.N3+i]

				var bit uint
				if bytes.Compare(label, wire.Label0.Bytes()) == 0 {
					bit = 0
				} else if bytes.Compare(label, wire.Label1.Bytes()) == 0 {
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
			conn.Flush()
			fmt.Printf("Result: %v\n", result)
			done = true
		}
	}
	t = time.Now()
	fmt.Printf("OT:\t%s\n", lastOT.Sub(start))
	fmt.Printf("Eval:\t%s\n", t.Sub(lastOT))
	start = t

	return nil
}

func evaluatorMode(circ *circuit.Circuit, input *big.Int) error {
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	defer nc.Close()

	conn := bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc))

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

	wires := make(map[circuit.Wire]*ot.Label)

	// Receive peer inputs.
	for i := 0; i < circ.N1; i++ {
		n, err := receiveData(conn)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Printf("N1[%d]:\t%x\n", i, n)
		}
		wires[circuit.Wire(i)] = ot.LabelFromData(n)
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
		wires[circuit.Wire(circ.N1+i)] = ot.LabelFromData(n)
	}

	// Evaluate gates.
	err = circ.Eval(key[:], wires, garbled)
	if err != nil {
		return err
	}

	var labels []*ot.Label

	for i := 0; i < circ.N3; i++ {
		r := wires[circuit.Wire(circ.NumWires-circ.N3+i)]
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

func receive(conn *bufio.ReadWriter, receiver *ot.Receiver, wire, bit int) (
	[]byte, error) {

	if err := sendUint32(conn, OP_OT); err != nil {
		return nil, err
	}
	if err := sendUint32(conn, wire); err != nil {
		return nil, err
	}
	conn.Flush()

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
	conn.Flush()

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

func result(conn *bufio.ReadWriter, labels []*ot.Label) (*big.Int, error) {
	if err := sendUint32(conn, OP_RESULT); err != nil {
		return nil, err
	}
	for _, l := range labels {
		if err := sendData(conn, l.Bytes()); err != nil {
			return nil, err
		}
	}
	conn.Flush()

	result, err := receiveData(conn)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetBytes(result), nil
}
