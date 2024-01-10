//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"github.com/markkurossi/mpc/ot"
)

// FxSend implements the sender part of the secure multiplication
// algorithm Fx (3.1.1 Secure Multiplication - page 7).
func FxSend(oti ot.OT, a Label) (r Label, err error) {
	r, err = NewLabel()
	if err != nil {
		return
	}

	x0 := r
	x1 := r
	x1.Xor(a)

	wire := ot.Wire{
		L0: x0.ToOT(),
		L1: x1.ToOT(),
	}

	err = oti.Send([]ot.Wire{wire})
	return
}

// FxReceive implements the receiver part of the secure multiplication
// algorithm Fx (3.1.1 Secure Multiplication - page 7).
func FxReceive(oti ot.OT, bit bool) (xb Label, err error) {
	flags := []bool{bit}
	var result [1]ot.Label

	err = oti.Receive(flags, result[:])
	if err != nil {
		return
	}
	xb.FromOT(result[0])
	return
}
