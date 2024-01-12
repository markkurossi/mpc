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
func FxSend(oti ot.OT, a uint) (r uint, err error) {
	rl, err := NewLabel()
	if err != nil {
		return 0, err
	}

	x0 := rl
	x1 := rl

	var al Label
	al[0] = byte(a)
	x1.Xor(al)

	wire := ot.Wire{
		L0: x0.ToOT(),
		L1: x1.ToOT(),
	}

	err = oti.Send([]ot.Wire{wire})
	if err != nil {
		return 0, err
	}
	return uint(rl[0] & 1), nil
}

// FxReceive implements the receiver part of the secure multiplication
// algorithm Fx (3.1.1 Secure Multiplication - page 7).
func FxReceive(oti ot.OT, b uint) (xb uint, err error) {
	flags := []bool{b == 1}
	var result [1]ot.Label

	err = oti.Receive(flags, result[:])
	if err != nil {
		return 0, err
	}
	var xl Label
	xl.FromOT(result[0])
	return uint(xl[0] & 1), nil
}

// FxkSend implements the sender part of the secure multiplication
// algorithm Fx (3.1.1 Secure Multiplication - page 7).
func FxkSend(oti ot.OT, s Label) (r Label, err error) {
	r, err = NewLabel()
	if err != nil {
		return
	}

	x0 := r
	x1 := r
	x1.Xor(s)

	wire := ot.Wire{
		L0: x0.ToOT(),
		L1: x1.ToOT(),
	}

	err = oti.Send([]ot.Wire{wire})
	return
}

// FxkReceive implements the receiver part of the secure multiplication
// algorithm Fx (3.1.1 Secure Multiplication - page 7).
func FxkReceive(oti ot.OT, b uint) (xb Label, err error) {
	flags := []bool{b == 1}
	var result [1]ot.Label

	err = oti.Receive(flags, result[:])
	if err != nil {
		return
	}
	xb.FromOT(result[0])
	return
}
