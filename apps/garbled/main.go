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
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
)

const (
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
			circ, err = compiler.CompileFile(arg)
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

		result, err := circuit.Garbler(io, circ, input, key[:], verbose)

		conn.Close()

		if err != nil {
			return err
		}

		printResult(result)
	}
	return nil
}

func evaluatorMode(circ *circuit.Circuit, input *big.Int) error {
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	defer nc.Close()

	conn := bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc))

	result, err := circuit.Evaluator(conn, circ, input, key[:])
	if err != nil {
		return err
	}
	printResult(result)

	return nil
}

func printResult(result *big.Int) {
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("Result: 0b%s\n", result.Text(2))
	fmt.Printf("Result: 0x%x\n", result.Bytes())
}
