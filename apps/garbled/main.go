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
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
)

const (
	OP_OT = iota
	OP_RESULT
)

var (
	port    = ":8080"
	verbose = false
	debug   = false
	key     [32]byte // XXX
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
	evaluator := flag.Bool("e", false, "evaluator / garbler mode")
	compile := flag.Bool("circ", false, "compile MPCL to circuit")
	ssa := flag.Bool("ssa", false, "compile MPCL to SSA assembly")
	dot := flag.Bool("dot", false, "create Graphviz DOT output")
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

	params := &utils.Params{
		Verbose: *fVerbose,
	}

	for _, arg := range flag.Args() {
		if strings.HasSuffix(arg, ".circ") {
			circ, err = loadCircuit(arg)
			if err != nil {
				fmt.Printf("Failed to parse circuit file '%s': %s\n", arg, err)
				os.Exit(1)
			}
		} else if strings.HasSuffix(arg, ".mpcl") {
			if *ssa {
				params.SSAOut, err = makeOutput(arg, "ssa")
				if err != nil {
					fmt.Printf("Failed to create SSA file: %s\n", err)
					os.Exit(1)
				}
				if *dot {
					params.SSADotOut, err = makeOutput(arg, "ssa.dot")
					if err != nil {
						fmt.Printf("Failed to create SSA DOT file: %s\n", err)
						os.Exit(1)
					}
				}
			}
			if *compile {
				params.CircOut, err = makeOutput(arg, "circ")
				if err != nil {
					fmt.Printf("Failed to create circuit file: %s\n", err)
					os.Exit(1)
				}
				if *dot {
					params.CircDotOut, err = makeOutput(arg, "circ.dot")
					if err != nil {
						fmt.Printf("Failed to create circuit DOT file: %s\n",
							err)
						os.Exit(1)
					}
				}
			}

			circ, err = compiler.CompileFile(arg, params)
			if err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}
			params.Close()
		} else {
			fmt.Printf("Unknown file type '%s'\n", arg)
			os.Exit(1)
		}
	}

	if *ssa || *compile {
		return
	}

	fmt.Printf("Circuit: %v\n", circ)
	var n1t, n2t string
	if *evaluator {
		n1t = "- "
		n2t = "+ "
	} else {
		n1t = "+ "
		n2t = "- "
	}

	fmt.Printf(" %sN1: %s\n", n1t, circ.N1)
	fmt.Printf(" %sN2: %s\n", n2t, circ.N2)
	fmt.Printf(" - N3: %s\n", circ.N3)
	fmt.Printf(" - In: %s\n", inputFlag)

	var input []*big.Int
	if *evaluator {
		input, err = circ.N2.Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = evaluatorMode(circ, input)
	} else {
		input, err = circ.N1.Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = garblerMode(circ, input)
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

func evaluatorMode(circ *circuit.Circuit, input []*big.Int) error {
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

		bio := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		ccon := circuit.NewConn(bio)

		result, err := circuit.Evaluator(ccon, circ, input, key[:], verbose)

		conn.Close()

		if err != nil && err != io.EOF {
			return err
		}

		printResult(result)
	}
}

func garblerMode(circ *circuit.Circuit, input []*big.Int) error {
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	defer nc.Close()

	bio := bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc))
	conn := circuit.NewConn(bio)

	result, err := circuit.Garbler(conn, circ, input, key[:], verbose)
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
		bytes := result.Bytes()
		if len(bytes) == 0 {
			bytes = []byte{0}
		}
		fmt.Printf("Result[%d]: 0x%x\n", idx, bytes)
	}
}

func makeOutput(base, suffix string) (*os.File, error) {
	var path string

	idx := strings.LastIndexByte(base, '.')
	if idx < 0 {
		path = base + "." + suffix
	} else {
		path = base[:idx+1] + suffix
	}
	return os.Create(path)
}
