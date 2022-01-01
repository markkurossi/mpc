//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"io"
)

// Params specify compiler parameters.
type Params struct {
	Verbose     bool
	Diagnostics bool
	SSAOut      io.WriteCloser
	SSADotOut   io.WriteCloser

	// MaxVarBits specifies the maximum variable width in bits.
	MaxVarBits int

	// MaxLoopUnroll specifies the upper limit for loop unrolling.
	MaxLoopUnroll int

	NoCircCompile bool
	CircOut       io.WriteCloser
	CircDotOut    io.WriteCloser
	CircFormat    string
	CircAnalyze   bool

	CircMultArrayTreshold int

	OptPruneGates bool
}

// NewParams returns new compiler params object, initialized with the
// default values.
func NewParams() *Params {
	return &Params{
		MaxVarBits:    0x20000,
		MaxLoopUnroll: 0x20000,
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
}
