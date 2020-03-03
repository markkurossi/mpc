//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"io"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
)

func Compile(data string, params *utils.Params) (*circuit.Circuit, error) {
	return compile("{data}", strings.NewReader(data), params)
}

func CompileFile(file string, params *utils.Params) (*circuit.Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return compile(file, f, params)
}

func compile(name string, in io.Reader, params *utils.Params) (
	*circuit.Circuit, error) {

	logger := utils.NewLogger(name, os.Stdout)
	parser := NewParser(logger, in)
	unit, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return unit.Compile(logger, params)
}
