//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"io"
)

type Params struct {
	Verbose    bool
	SSAOut     io.WriteCloser
	SSADotOut  io.WriteCloser
	CircOut    io.WriteCloser
	CircDotOut io.WriteCloser

	CircMultArrayTreshold int

	OptPruneGates bool
}

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
