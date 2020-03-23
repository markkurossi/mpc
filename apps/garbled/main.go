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
	"runtime/pprof"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/p2p"
)

const (
	OP_OT = iota
	OP_RESULT
)

var (
	port    = ":8080"
	verbose = false
	debug   = false
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
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	bmr := flag.Int("bmr", -1, "semi-honest secure BMR protocol player number")
	flag.Parse()

	verbose = *fVerbose
	debug = *fDebug

	var circ *circuit.Circuit
	var err error

	if len(flag.Args()) == 0 {
		fmt.Printf("No input files\n")
		os.Exit(1)
	}

	if len(*cpuprofile) > 0 {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	params := &utils.Params{
		Verbose: *fVerbose,
	}
	defer params.Close()

	for _, arg := range flag.Args() {
		if strings.HasSuffix(arg, ".circ") {
			circ, err = loadCircuit(arg)
			if err != nil {
				fmt.Printf("Failed to parse circuit file '%s': %s\n", arg, err)
				return
			}
		} else if strings.HasSuffix(arg, ".mpcl") {
			if *ssa {
				params.SSAOut, err = makeOutput(arg, "ssa")
				if err != nil {
					fmt.Printf("Failed to create SSA file: %s\n", err)
					return
				}
				if *dot {
					params.SSADotOut, err = makeOutput(arg, "ssa.dot")
					if err != nil {
						fmt.Printf("Failed to create SSA DOT file: %s\n", err)
						return
					}
				}
			}
			if *compile {
				params.CircOut, err = makeOutput(arg, "circ")
				if err != nil {
					fmt.Printf("Failed to create circuit file: %s\n", err)
					return
				}
				if *dot {
					params.CircDotOut, err = makeOutput(arg, "circ.dot")
					if err != nil {
						fmt.Printf("Failed to create circuit DOT file: %s\n",
							err)
						return
					}
				}
			}

			circ, _, err = compiler.NewCompiler(params).CompileFile(arg)
			if err != nil {
				fmt.Printf("%s\n", err)
				return
			}
		} else {
			fmt.Printf("Unknown file type '%s'\n", arg)
			return
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

	if *bmr >= 0 {
		fmt.Printf("semi-honest secure BMR protocol\n")
		fmt.Printf("player: %d\n", *bmr)

		for _, flag := range inputFlag {
			i := new(big.Int)
			_, ok := i.SetString(flag, 0)
			if !ok {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}
			input = append(input, i)
		}
		err := bmrMode(circ, input, *bmr)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		return
	}

	if *evaluator {
		input, err = circ.N2.Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = evaluatorMode(circ, input, len(*cpuprofile) > 0)
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

func evaluatorMode(circ *circuit.Circuit, input []*big.Int, once bool) error {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	fmt.Printf("Listening for connections at %s\n", port)

	for {
		nc, err := ln.Accept()
		if err != nil {
			return err
		}
		fmt.Printf("New connection from %s\n", nc.RemoteAddr())

		conn := p2p.NewConn(nc)
		result, err := circuit.Evaluator(conn, circ, input, verbose)
		conn.Close()

		if err != nil && err != io.EOF {
			return err
		}

		printResult(result)
		if once {
			return nil
		}
	}
}

func garblerMode(circ *circuit.Circuit, input []*big.Int) error {
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(nc)
	defer conn.Close()

	result, err := circuit.Garbler(conn, circ, input, verbose)
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

func makeOutput(base, suffix string) (io.WriteCloser, error) {
	var path string

	idx := strings.LastIndexByte(base, '.')
	if idx < 0 {
		path = base + "." + suffix
	} else {
		path = base[:idx+1] + suffix
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &OutputFile{
		File:     f,
		Buffered: bufio.NewWriter(f),
	}, nil
}

type OutputFile struct {
	File     *os.File
	Buffered *bufio.Writer
}

func (out *OutputFile) Write(p []byte) (nn int, err error) {
	return out.Buffered.Write(p)
}

func (out *OutputFile) Close() error {
	if err := out.Buffered.Flush(); err != nil {
		return err
	}
	return out.File.Close()
}
