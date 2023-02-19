//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

func streamEvaluatorMode(oti ot.OT, input input, once bool) error {
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
		outputs, result, err := circuit.StreamEvaluator(conn, oti, input,
			verbose)
		conn.Close()

		if err != nil && err != io.EOF {
			return err
		}

		printResults(result, outputs)
		if once {
			return nil
		}
	}
}

func streamGarblerMode(params *utils.Params, oti ot.OT, input input,
	args []string) error {

	if len(args) != 1 || !strings.HasSuffix(args[0], ".mpcl") {
		return fmt.Errorf("streaming mode takes single MPCL file")
	}
	nc, err := net.Dial("tcp", port)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(nc)
	defer conn.Close()

	outputs, result, err := compiler.New(params).StreamFile(
		conn, oti, args[0], input)
	if err != nil {
		return err
	}
	printResults(result, outputs)
	return nil
}
