//
// Copyright (c) 2020-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"net"

	"github.com/markkurossi/mpc"
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

func streamEvaluatorMode(oti ot.OT, input input, once bool) error {
	inputSizes, err := circuit.InputSizes(input)
	if err != nil {
		return err
	}

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

		err = conn.SendInputSizes(inputSizes)
		if err != nil {
			conn.Close()
			return err
		}
		err = conn.Flush()
		if err != nil {
			conn.Close()
			return err
		}

		outputs, result, err := circuit.StreamEvaluator(conn, oti, input, nil,
			verbose)
		conn.Close()

		if err != nil && err != io.EOF {
			return fmt.Errorf("%s: %v", nc.RemoteAddr(), err)
		}

		mpc.PrintResults(result, outputs, base)
		if once {
			return nil
		}
	}
}

func streamGarblerMode(params *utils.Params, oti ot.OT, input input,
	args []string) error {

	inputSizes := make([][]int, 2)

	sizes, err := circuit.InputSizes(input)
	if err != nil {
		return err
	}
	inputSizes[0] = sizes

	if len(args) != 1 {
		return fmt.Errorf("streaming mode takes single MPCL file")
	}
	if !compiler.IsFilename(args[0]) {
		return fmt.Errorf("unsupported file for streaming: %v", args[0])
	}
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(nc)
	defer conn.Close()

	sizes, err = conn.ReceiveInputSizes()
	if err != nil {
		return err
	}
	inputSizes[1] = sizes

	outputs, result, err := compiler.New(params).StreamFile(
		conn, oti, args[0], input, inputSizes)
	if err != nil {
		return err
	}
	mpc.PrintResults(result, outputs, base)
	return nil
}
