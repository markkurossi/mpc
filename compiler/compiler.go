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

type Compiler struct {
	params *utils.Params
}

func NewCompiler(params *utils.Params) *Compiler {
	return &Compiler{
		params: params,
	}
}

func (c *Compiler) Compile(data string) (*circuit.Circuit, error) {
	return c.compile("{data}", strings.NewReader(data))
}

func (c *Compiler) CompileFile(file string) (*circuit.Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return c.compile(file, f)
}

func (c *Compiler) compile(name string, in io.Reader) (
	*circuit.Circuit, error) {

	logger := utils.NewLogger(name, os.Stdout)

	parser := NewParser(c, logger, in)
	unit, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return unit.Compile(logger, c.params)
}
