//
// main.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

var (
	port    = ":8080"
	verbose = false
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
	stream := flag.Bool("stream", false, "streaming mode")
	compile := flag.Bool("circ", false, "compile MPCL to circuit")
	circFormat := flag.String("format", "mpclc",
		"circuit format: mpclc, bristol")
	ssa := flag.Bool("ssa", false, "compile MPCL to SSA assembly")
	dot := flag.Bool("dot", false, "create Graphviz DOT output")
	svg := flag.Bool("svg", false, "create SVG output")
	optimize := flag.Int("O", 1, "optimization level")
	fVerbose := flag.Bool("v", false, "verbose output")
	fDiagnostics := flag.Bool("d", false, "diagnostics output")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "",
		"write memory profile to `file`")
	bmr := flag.Int("bmr", -1, "semi-honest secure BMR protocol player number")
	flag.Parse()

	log.SetFlags(0)

	verbose = *fVerbose

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

	params := utils.NewParams()
	defer params.Close()

	params.Verbose = *fVerbose
	params.Diagnostics = *fDiagnostics

	if *optimize > 0 {
		params.OptPruneGates = true
	}
	if *ssa && !*compile {
		params.NoCircCompile = true
	}

	if *compile || *ssa {
		err := compileFiles(flag.Args(), params, *compile, *ssa, *dot, *svg,
			*circFormat)
		if err != nil {
			log.Fatalf("compile failed: %s", err)
		}
		memProfile(*memprofile)
		return
	}

	var err error

	oti := ot.NewCO()

	if *stream {
		if *evaluator {
			err = streamEvaluatorMode(oti, inputFlag, len(*cpuprofile) > 0)
		} else {
			err = streamGarblerMode(params, oti, inputFlag, flag.Args())
		}
		memProfile(*memprofile)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		return
	}

	if len(flag.Args()) != 1 {
		fmt.Printf("expected one input file, got %v\n", len(flag.Args()))
		os.Exit(1)
	}

	var circ *circuit.Circuit

	file := flag.Args()[0]
	if circuit.IsFilename(file) {
		circ, err = circuit.Parse(file)
		if err != nil {
			fmt.Printf("failed to parse circuit file '%s': %s\n", file, err)
			return
		}
	} else if strings.HasSuffix(file, ".mpcl") {
		circ, _, err = compiler.New(params).CompileFile(file)
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
	} else {
		fmt.Printf("unknown file type '%s'\n", file)
		return
	}

	if circ != nil {
		circ.AssignLevels()
		if verbose {
			fmt.Printf("circuit: %v\n", circ)
		}
	}

	var input *big.Int

	if *bmr >= 0 {
		fmt.Printf("semi-honest secure BMR protocol\n")
		fmt.Printf("player: %d\n", *bmr)

		if *bmr >= len(circ.Inputs) {
			fmt.Printf("invalid party number %d for %d-party computation\n",
				*bmr, len(circ.Inputs))
			return
		}

		input, err = circ.Inputs[*bmr].Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		for idx, arg := range circ.Inputs {
			if idx == *bmr {
				fmt.Printf(" + In%d: %s\n", idx, arg)
			} else {
				fmt.Printf(" - In%d: %s\n", idx, arg)
			}
		}

		fmt.Printf(" - Out: %s\n", circ.Outputs)
		fmt.Printf(" - In:  %s\n", inputFlag)

		err := bmrMode(circ, input, *bmr)
		if err != nil {
			fmt.Printf("BMR mode failed: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if len(circ.Inputs) != 2 {
		fmt.Printf("invalid circuit for 2-party computation: %d parties\n",
			len(circ.Inputs))
		return
	}

	var i1t, i2t string
	if *evaluator {
		i1t = "- "
		i2t = "+ "
	} else {
		i1t = "+ "
		i2t = "- "
	}

	fmt.Printf(" %sIn1: %s\n", i1t, circ.Inputs[0])
	fmt.Printf(" %sIn2: %s\n", i2t, circ.Inputs[1])
	fmt.Printf(" - Out: %s\n", circ.Outputs)
	fmt.Printf(" -  In: %s\n", inputFlag)

	if *evaluator {
		input, err = circ.Inputs[1].Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = evaluatorMode(oti, circ, input, len(*cpuprofile) > 0)
	} else {
		input, err = circ.Inputs[0].Parse(inputFlag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		err = garblerMode(oti, circ, input)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func memProfile(file string) {
	if len(file) == 0 {
		return
	}

	f, err := os.Create(file)
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer f.Close()
	if false {
		runtime.GC()
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}

func evaluatorMode(oti ot.OT, circ *circuit.Circuit, input *big.Int,
	once bool) error {
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
		result, err := circuit.Evaluator(conn, oti, circ, input, verbose)
		conn.Close()

		if err != nil && err != io.EOF {
			return err
		}

		printResults(result, circ.Outputs)
		if once {
			return nil
		}
	}
}

func garblerMode(oti ot.OT, circ *circuit.Circuit, input *big.Int) error {
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(nc)
	defer conn.Close()

	result, err := circuit.Garbler(conn, oti, circ, input, verbose)
	if err != nil {
		return err
	}
	printResults(result, circ.Outputs)

	return nil
}
