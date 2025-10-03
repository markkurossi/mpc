//
// main.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
)

func compileFiles(files []string, params *utils.Params, inputSizes [][]int,
	compile, ssa, dot, svg bool, circSuffix string) error {

	var circ *circuit.Circuit
	var err error

	for _, file := range files {
		if compile {
			params.CircOut, err = makeOutput(file, circSuffix)
			if err != nil {
				return err
			}
			if dot {
				params.CircDotOut, err = makeOutput(file, "circ.dot")
				if err != nil {
					return err
				}
			}
			if svg {
				params.CircSvgOut, err = makeOutput(file, "circ.svg")
				if err != nil {
					return err
				}
			}
		}
		if circuit.IsFilename(file) {
			circ, err = circuit.Parse(file)
			if err != nil {
				return err
			}
			if params.CircOut != nil {
				if params.Verbose {
					fmt.Printf("Serializing circuit...\n")
				}
				err = circ.MarshalFormat(params.CircOut, params.CircFormat)
				if err != nil {
					return err
				}
			}
		} else if compiler.IsFilename(file) {
			if ssa {
				params.SSAOut, err = makeOutput(file, "ssa")
				if err != nil {
					return err
				}
				if dot {
					params.SSADotOut, err = makeOutput(file, "ssa.dot")
					if err != nil {
						return err
					}
				}
			}
			_, _, err = compiler.New(params).CompileFile(file, inputSizes)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown file type '%s'", file)
		}
	}
	return nil
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

// OutputFile implements a buffered output file.
type OutputFile struct {
	File     *os.File
	Buffered *bufio.Writer
}

func (out *OutputFile) Write(p []byte) (nn int, err error) {
	return out.Buffered.Write(p)
}

// Close implements io.Closer.Close for the buffered output file.
func (out *OutputFile) Close() error {
	if err := out.Buffered.Flush(); err != nil {
		return err
	}
	return out.File.Close()
}
