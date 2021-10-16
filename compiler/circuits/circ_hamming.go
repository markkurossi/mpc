//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/types"
)

// Hamming creates a hamming distance circuit computing the hamming
// distance between a and b and returning the distance in r.
func Hamming(compiler *Compiler, a, b, r []*Wire) error {
	a, b = compiler.ZeroPad(a, b)

	var arr [][]*Wire
	for i := 0; i < len(a); i++ {
		w := NewWire()
		compiler.AddGate(NewBinary(circuit.XOR, a[i], b[i], w))
		arr = append(arr, []*Wire{w})
	}

	for len(arr) > 2 {
		var n [][]*Wire
		for i := 0; i < len(arr); i += 2 {
			if i+1 < len(arr) {
				result := MakeWires(types.Size(len(arr[i]) + 1))
				err := NewAdder(compiler, arr[i], arr[i+1], result)
				if err != nil {
					return err
				}
				n = append(n, result)
			} else {
				n = append(n, arr[i])
			}
		}
		arr = n
	}

	return NewAdder(compiler, arr[0], arr[1], r)
}
