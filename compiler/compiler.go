//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Compiler struct {
	params   *utils.Params
	packages map[string]*ast.Package
}

func NewCompiler(params *utils.Params) *Compiler {
	return &Compiler{
		params:   params,
		packages: make(map[string]*ast.Package),
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
	unit, err := c.parse(name, in, logger)
	if err != nil {
		return nil, err
	}

	return unit.Compile(c.packages, logger, c.params)
}

func (c *Compiler) parse(name string, in io.Reader, logger *utils.Logger) (
	*ast.Package, error) {
	parser := NewParser(c, logger, in)
	pkg, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	c.packages[pkg.Name] = pkg

	for _, ref := range pkg.References {
		c.parsePkg(ref)
	}

	return pkg, nil
}

func (c *Compiler) parsePkg(name string) error {
	_, ok := c.packages[name]
	if ok {
		return nil
	}

	fmt.Printf("Looking for package %s\n", name)
	prefix := "go/src/github.com/markkurossi/mpc/pkg"
	expanded := path.Join(os.Getenv("HOME"), prefix,
		fmt.Sprintf("%s/integer.mpcl", name))
	fmt.Printf("=> %s\n", expanded)

	f, err := os.Open(expanded)
	if err != nil {
		fmt.Printf("pkg not found: %s\n", err)
		return fmt.Errorf("package %s not found", name)
	}
	defer f.Close()

	_, err = c.parse(expanded, f, utils.NewLogger(expanded, os.Stdout))
	return err
}
