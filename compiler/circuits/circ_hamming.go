//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
)

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
				result := makeWires(len(arr[i]) + 1)
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

func makeWires(count int) []*Wire {
	result := make([]*Wire, count)
	for i := 0; i < count; i++ {
		result[i] = NewWire()
	}
	return result
}
