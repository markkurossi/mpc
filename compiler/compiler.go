//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/p2p"
)

// Compiler implements MPCL compiler.
type Compiler struct {
	params   *utils.Params
	packages map[string]*ast.Package
}

// New creates a new compiler instance.
func New(params *utils.Params) *Compiler {
	return &Compiler{
		params:   params,
		packages: make(map[string]*ast.Package),
	}
}

// Compile compiles the input program.
func (c *Compiler) Compile(data string) (*circuit.Circuit, ast.Annotations,
	error) {
	return c.compile("{data}", strings.NewReader(data))
}

// CompileFile compiles the input file.
func (c *Compiler) CompileFile(file string) (*circuit.Circuit, ast.Annotations,
	error) {

	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	return c.compile(file, f)
}

// ParseFile parses the input file.
func (c *Compiler) ParseFile(file string) (*ast.Package, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	logger := utils.NewLogger(os.Stdout)
	return c.parse(file, f, logger, nil)
}

func (c *Compiler) compile(source string, in io.Reader) (
	*circuit.Circuit, ast.Annotations, error) {

	logger := utils.NewLogger(os.Stdout)
	pkg, err := c.parse(source, in, logger, ast.NewPackage("main", source, nil))
	if err != nil {
		return nil, nil, err
	}

	ctx := ast.NewCodegen(logger, pkg, c.packages, c.params)

	program, annotation, err := pkg.Compile(ctx)
	if err != nil {
		return nil, nil, err
	}
	if c.params.NoCircCompile {
		return nil, annotation, nil
	}
	circ, err := program.CompileCircuit(c.params)
	if err != nil {
		return nil, nil, err
	}
	return circ, annotation, nil
}

// StreamFile compiles the input program and uses the streaming mode
// to garble and stream the circuit to the evaluator node.
func (c *Compiler) StreamFile(conn *p2p.Conn, file string,
	input []string) (circuit.IO, []*big.Int, error) {

	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	return c.stream(conn, file, f, input)
}

func (c *Compiler) stream(conn *p2p.Conn, source string, in io.Reader,
	inputFlag []string) (circuit.IO, []*big.Int, error) {

	logger := utils.NewLogger(os.Stdout)
	pkg, err := c.parse(source, in, logger, ast.NewPackage("main", source, nil))
	if err != nil {
		return nil, nil, err
	}

	ctx := ast.NewCodegen(logger, pkg, c.packages, c.params)

	program, _, err := pkg.Compile(ctx)
	if err != nil {
		return nil, nil, err
	}

	if len(program.Inputs) != 2 {
		return nil, nil,
			fmt.Errorf("invalid program for 2-party computation: %d parties",
				len(program.Inputs))
	}
	input, err := program.Inputs[0].Parse(inputFlag)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf(" + In1: %s\n", program.Inputs[0])
	fmt.Printf(" - In2: %s\n", program.Inputs[1])
	fmt.Printf(" - Out: %s\n", program.Outputs)
	fmt.Printf(" -  In: %s\n", inputFlag)

	return program.StreamCircuit(conn, c.params, input)
}

func (c *Compiler) parse(source string, in io.Reader, logger *utils.Logger,
	pkg *ast.Package) (*ast.Package, error) {

	parser := NewParser(source, c, logger, in)
	pkg, err := parser.Parse(pkg)
	if err != nil {
		return nil, err
	}
	c.packages[pkg.Name] = pkg

	for alias, name := range pkg.Imports {
		_, err := c.parsePkg(alias, name, source)
		if err != nil {
			return nil, err
		}
	}

	return pkg, nil
}

type packagePath struct {
	precond string
	env     string
	prefix  string
}

var packagePaths = []packagePath{
	{
		env:    "HOME",
		prefix: "go/src/github.com/markkurossi/mpc/pkg",
	},
	{
		precond: "GITHUB_WORKFLOW",
		env:     "GITHUB_WORKSPACE",
		prefix:  "pkg",
	},
}

func (c *Compiler) parsePkg(alias, name, source string) (*ast.Package, error) {
	pkg, ok := c.packages[alias]
	if ok {
		return pkg, nil
	}
	pkg = ast.NewPackage(alias, source, nil)

	if c.params.Verbose {
		fmt.Printf("looking for package %s (%s)\n", alias, name)
	}

	var dir string
	var df *os.File
	var err error
	for _, pkgPath := range packagePaths {
		// Check path precondition.
		if len(pkgPath.precond) > 0 {
			_, ok := os.LookupEnv(pkgPath.precond)
			if !ok {
				continue
			}
		}
		dir = path.Join(os.Getenv(pkgPath.env), pkgPath.prefix, name)
		df, err = os.Open(dir)
		if err == nil {
			defer df.Close()
			break
		}
	}
	if df == nil {
		return nil, fmt.Errorf("package %s not found", name)
	}

	files, err := df.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("package %s not found: %s", name, err)
	}
	var mpcls []string
	for _, f := range files {
		if strings.HasSuffix(f, ".mpcl") {
			mpcls = append(mpcls, f)
		}
	}
	if len(mpcls) == 0 {
		return nil, fmt.Errorf("package %s is empty", name)
	}

	for _, mpcl := range mpcls {
		fp := path.Join(dir, mpcl)

		if c.params.Verbose {
			fmt.Printf(" - parsing %v\n", fp)
		}

		f, err := os.Open(fp)
		if err != nil {
			fmt.Printf("pkg not found: %s\n", err)
			return nil, fmt.Errorf("error reading package %s: %s", name, err)
		}
		defer f.Close()

		pkg, err = c.parse(fp, f, utils.NewLogger(os.Stdout), pkg)
		if err != nil {
			return nil, err
		}
	}
	return pkg, nil
}
