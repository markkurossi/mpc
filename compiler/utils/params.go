//
// Copyright (c) 2020-2025 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"io"
)

// Params specify compiler parameters.
type Params struct {
	Verbose       bool
	Diagnostics   bool
	SSAOut        io.WriteCloser
	SSADotOut     io.WriteCloser
	MPCLCErrorLoc bool

	// PkgPath defines additional directories to search for imported
	// packages.
	PkgPath []string

	// MaxLoopUnroll specifies the upper limit for loop unrolling.
	MaxLoopUnroll int

	NoCircCompile bool
	CircOut       io.WriteCloser
	CircDotOut    io.WriteCloser
	CircSvgOut    io.WriteCloser
	CircFormat    string

	CircMultArrayTreshold int

	OptPruneGates bool

	BenchmarkCompile bool

	// SymbolIDs contain the mappings from the interned symbols to
	// their values. You must persist these in the application code if
	// you want to maintain the mappings across different compiler
	// invocations.
	SymbolIDs map[string]int
}

// NewParams returns new compiler params object, initialized with the
// default values.
func NewParams() *Params {
	return &Params{
		MaxLoopUnroll: 0x20000,
		SymbolIDs:     make(map[string]int),
	}
}

// Close closes all open resources.
func (p *Params) Close() {
	if p.SSAOut != nil {
		p.SSAOut.Close()
		p.SSAOut = nil
	}
	if p.SSADotOut != nil {
		p.SSADotOut.Close()
		p.SSADotOut = nil
	}
	if p.CircOut != nil {
		p.CircOut.Close()
		p.CircOut = nil
	}
	if p.CircDotOut != nil {
		p.CircDotOut.Close()
		p.CircDotOut = nil
	}
	if p.CircSvgOut != nil {
		p.CircSvgOut.Close()
		p.CircSvgOut = nil
	}
}
