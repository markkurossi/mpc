//
// main.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"
	"strings"

	"github.com/markkurossi/mpc"
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

var inputFlag, peerFlag input

func init() {
	flag.Var(&inputFlag, "i", "comma-separated list of circuit inputs")
	flag.Var(&peerFlag, "pi", "comma-separated list of peer's circuit inputs")
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
	mpclcErrLoc := flag.Bool("mpclc-err-loc", false,
		"print MPCLC error locations")
	benchmarkCompile := flag.Bool("benchmark-compile", false,
		"benchmark MPCL compilation")
	sids := flag.String("sids", "", "store symbol IDs `file`")
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
	params.MPCLCErrorLoc = *mpclcErrLoc
	params.BenchmarkCompile = *benchmarkCompile

	if *optimize > 0 {
		params.OptPruneGates = true
	}
	if *ssa && !*compile {
		params.NoCircCompile = true
	}

	if len(*sids) > 0 {
		err := params.LoadSymbolIDs(*sids)
		if err != nil {
			log.Fatalf("failed to load symbol IDs: %v", err)
		}
	}

	if *compile || *ssa {
		inputSizes := make([][]int, 2)
		iSizes, err := circuit.InputSizes(inputFlag)
		if err != nil {
			log.Fatal(err)
		}
		pSizes, err := circuit.InputSizes(peerFlag)
		if err != nil {
			log.Fatal(err)
		}
		if *evaluator {
			inputSizes[0] = pSizes
			inputSizes[1] = iSizes
		} else {
			inputSizes[0] = iSizes
			inputSizes[1] = pSizes
		}

		err = compileFiles(flag.Args(), params, inputSizes,
			*compile, *ssa, *dot, *svg, *circFormat)
		if err != nil {
			log.Fatalf("compile failed: %s", err)
		}
		memProfile(*memprofile)

		if len(*sids) > 0 {
			err = params.SaveSymbolIDs("main", *sids)
			if err != nil {
				log.Fatalf("failed to save symbol IDs: %v", err)
			}
		}
		return
	}

	var err error

	oti := ot.NewCO()

	if *stream {
		if *evaluator {
			err = streamEvaluatorMode(oti, inputFlag,
				len(*cpuprofile) > 0 || len(*memprofile) > 0)
		} else {
			err = streamGarblerMode(params, oti, inputFlag, flag.Args())
		}
		memProfile(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(flag.Args()) != 1 {
		log.Fatalf("expected one input file, got %v\n", len(flag.Args()))
	}
	file := flag.Args()[0]

	if *bmr >= 0 {
		err = bmrMode(file, params, *bmr)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if *evaluator {
		err = evaluatorMode(oti, file, params, len(*cpuprofile) > 0)
	} else {
		err = garblerMode(oti, file, params)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func loadCircuit(file string, params *utils.Params, inputSizes [][]int) (
	*circuit.Circuit, error) {

	var circ *circuit.Circuit
	var err error

	if circuit.IsFilename(file) {
		circ, err = circuit.Parse(file)
		if err != nil {
			return nil, err
		}
	} else if strings.HasSuffix(file, ".mpcl") {
		circ, _, err = compiler.New(params).CompileFile(file, inputSizes)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown file type '%s'", file)
	}

	if circ != nil {
		circ.AssignLevels()
		if verbose {
			fmt.Printf("circuit: %v\n", circ)
		}
	}
	return circ, err
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

func evaluatorMode(oti ot.OT, file string, params *utils.Params,
	once bool) error {

	inputSizes := make([][]int, 2)
	myInputSizes, err := circuit.InputSizes(inputFlag)
	if err != nil {
		return err
	}
	inputSizes[1] = myInputSizes

	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	fmt.Printf("Listening for connections at %s\n", port)

	var oPeerInputSizes []int
	var circ *circuit.Circuit

	for {
		nc, err := ln.Accept()
		if err != nil {
			return err
		}
		fmt.Printf("New connection from %s\n", nc.RemoteAddr())

		conn := p2p.NewConn(nc)

		err = conn.SendInputSizes(myInputSizes)
		if err != nil {
			conn.Close()
			return err
		}
		err = conn.Flush()
		if err != nil {
			conn.Close()
			return err
		}
		peerInputSizes, err := conn.ReceiveInputSizes()
		if err != nil {
			conn.Close()
			return err
		}
		inputSizes[0] = peerInputSizes

		if circ == nil || slices.Compare(peerInputSizes, oPeerInputSizes) != 0 {
			circ, err = loadCircuit(file, params, inputSizes)
			if err != nil {
				conn.Close()
				return err
			}
			oPeerInputSizes = peerInputSizes
		}
		circ.PrintInputs(circuit.IDEvaluator, inputFlag)
		if len(circ.Inputs) != 2 {
			return fmt.Errorf("invalid circuit for 2-party MPC: %d parties",
				len(circ.Inputs))
		}

		input, err := circ.Inputs[1].Parse(inputFlag)
		if err != nil {
			conn.Close()
			return fmt.Errorf("%s: %v", file, err)
		}
		result, err := circuit.Evaluator(conn, oti, circ, input, verbose)
		conn.Close()
		if err != nil && err != io.EOF {
			return err
		}
		mpc.PrintResults(result, circ.Outputs)
		if once {
			return nil
		}
	}
}

func garblerMode(oti ot.OT, file string, params *utils.Params) error {
	inputSizes := make([][]int, 2)
	myInputSizes, err := circuit.InputSizes(inputFlag)
	if err != nil {
		return err
	}
	inputSizes[0] = myInputSizes

	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(nc)
	defer conn.Close()

	peerInputSizes, err := conn.ReceiveInputSizes()
	if err != nil {
		conn.Close()
		return err
	}
	inputSizes[1] = peerInputSizes
	err = conn.SendInputSizes(myInputSizes)
	if err != nil {
		conn.Close()
		return err
	}
	err = conn.Flush()
	if err != nil {
		conn.Close()
		return err
	}

	circ, err := loadCircuit(file, params, inputSizes)
	if err != nil {
		return err
	}
	circ.PrintInputs(circuit.IDGarbler, inputFlag)
	if len(circ.Inputs) != 2 {
		return fmt.Errorf("invalid circuit for 2-party MPC: %d parties",
			len(circ.Inputs))
	}

	input, err := circ.Inputs[0].Parse(inputFlag)
	if err != nil {
		return fmt.Errorf("%s: %v", file, err)
	}
	result, err := circuit.Garbler(conn, oti, circ, input, verbose)
	if err != nil {
		return err
	}
	mpc.PrintResults(result, circ.Outputs)

	return nil
}
