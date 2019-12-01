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

type input []string

func (i *input) String() string {
	return fmt.Sprint(*i)
}

func (i *input) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		*i = append(*i, v)
	}
	return nil
}

var inputFlag input

func init() {
	flag.Var(&inputFlag, "i", "comma-separated list of circuit inputs")
}

func main() {
	garbler := flag.Bool("g", false, "garbler / evaluator mode")
	compile := flag.Bool("c", false, "compile MPCL to circuit")
	fVerbose := flag.Bool("v", false, "verbose output")
	fDebug := flag.Bool("d", false, "debug output")
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
				fmt.Printf("%s\n", err)
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
	fmt.Printf(" - N1: %s\n", circ.N1)
	fmt.Printf(" - N2: %s\n", circ.N2)
	fmt.Printf(" - N3: %s\n", circ.N3)
	fmt.Printf(" - In: %s\n", inputFlag)

	if *garbler {
		input, err := circ.N1.Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = garblerMode(circ, input)
	} else {
		input, err := circ.N2.Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = evaluatorMode(circ, input)
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

func garblerMode(circ *circuit.Circuit, input []*big.Int) error {
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

func evaluatorMode(circ *circuit.Circuit, input []*big.Int) error {
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

func printResult(results []*big.Int) {
	for idx, result := range results {
		fmt.Printf("Result[%d]: %v\n", idx, result)
		fmt.Printf("Result[%d]: 0b%s\n", idx, result.Text(2))
		fmt.Printf("Result[%d]: 0x%x\n", idx, result.Bytes())
	}
}
