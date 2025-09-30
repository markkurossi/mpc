//
// Copyright (c) 2021-2025 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
)

// NewIndex creates a new array element selection (index) circuit. The
// circuit uses n low-order bits from index to select the array
// value. The n is selected so that 2^n >= len(array)/size i.e. the
// circuit uses the minimum amount of bits from index that is needed
// to cover all array elements. The higher order bits are ignored
// i.e. the index is index % 2^n.
func NewIndex(cc *Compiler, size int, array, index, out []*Wire) error {
	if len(array)%size != 0 {
		return fmt.Errorf("array width %d must be multiple of element size %d",
			len(array), size)
	}
	if len(out) < size {
		return fmt.Errorf("out %d too small for element size %d",
			len(out), size)
	}
	n := len(array) / size
	if n == 0 {
		for i := 0; i < len(out); i++ {
			out[i] = cc.ZeroWire()
		}
		return nil
	}

	bits := 1
	var length int

	for length = 2; length < n; length *= 2 {
		bits++
	}

	// Default "not found" value.
	def := make([]*Wire, size)
	for i := 0; i < size; i++ {
		def[i] = cc.ZeroWire()
	}

	return newIndex(cc, bits-1, length, size, array, index, def, out)
}

func newIndex(cc *Compiler, bit, length, size int,
	array, index, def, out []*Wire) error {

	n := len(array) / size

	if bit == 0 {
		fVal := array[:size]

		var tVal []*Wire
		if n > 1 {
			tVal = array[size : 2*size]
		} else {
			tVal = def
		}
		return NewMUX(cc, index[0:1], tVal, fVal, out)
	}

	length /= 2
	fArray := array
	if n > length {
		fArray = fArray[:length*size]
	}

	if bit >= len(index) {
		// Not enough bits to select upper half so just select from
		// the lower half.
		return newIndex(cc, bit-1, length, size, fArray, index, def, out)
	}

	fVal := make([]*Wire, size)
	for i := 0; i < size; i++ {
		fVal[i] = cc.Calloc.Wire()
	}
	err := newIndex(cc, bit-1, length, size, fArray, index, def, fVal)
	if err != nil {
		return err
	}

	var tVal []*Wire
	if n > length {
		tVal = make([]*Wire, size)
		for i := 0; i < size; i++ {
			tVal[i] = cc.Calloc.Wire()
		}
		err = newIndex(cc, bit-1, length, size,
			array[length*size:], index, def, tVal)
		if err != nil {
			return err
		}
	} else {
		tVal = def
	}

	return NewMUX(cc, index[bit:bit+1], tVal, fVal, out)
}
