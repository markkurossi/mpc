//
// Copyright (c) 2020-2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/types"
)

// intComparator tests if x>y if cin=0, and x>=y if cin=1.
func intComparator(cc *Compiler, cin *Wire, x, y, r []*Wire) error {
	x, y = cc.ZeroPad(x, y)
	if len(x) == 0 {
		return fmt.Errorf("invalid int comparator arguments: len(x)=%d", len(x))
	}
	if len(r) != 1 {
		return fmt.Errorf("invalid int comparator arguments: len(r)=%d", len(r))
	}

	// Comparision when x and y have the same sign.

	var cout *Wire
	for i := 0; i < len(x); i++ {
		w1 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, cin, y[i], w1))
		w2 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, cin, x[i], w2))
		w3 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, w1, w2, w3))

		cout = cc.Calloc.Wire()

		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, cin, w3, cout))
		cin = cout
	}

	// If x and y have different sign, x is bigger if y is negative.

	signBit := len(y) - 1
	negBit := y[signBit]

	// Test if x and y have different sign i.e. x[signBig]^y[signBig]=1
	cond := cc.Calloc.Wire()
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[signBit], y[signBit], cond))

	// Result is negBit/cout based on cond=1/0
	return NewMUX(cc, []*Wire{cond}, []*Wire{negBit}, []*Wire{cout}, r)
}

// uintComparator tests if x>y if cin=0, and x>=y if cin=1.
func uintComparator(cc *Compiler, cin *Wire, x, y, r []*Wire) error {
	x, y = cc.ZeroPad(x, y)
	if len(r) != 1 {
		return fmt.Errorf("invalid uint comparator arguments: r=%d", len(r))
	}

	for i := 0; i < len(x); i++ {
		w1 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, cin, y[i], w1))
		w2 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, cin, x[i], w2))
		w3 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, w1, w2, w3))

		var cout *Wire
		if i+1 < len(x) {
			cout = cc.Calloc.Wire()
		} else {
			cout = r[0]
		}
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, cin, w3, cout))
		cin = cout
	}
	return nil
}

// NewIntGtComparator tests if x>y.
func NewIntGtComparator(cc *Compiler, x, y, r []*Wire) error {
	return intComparator(cc, cc.ZeroWire(), x, y, r)
}

// NewUintGtComparator tests if x>y.
func NewUintGtComparator(cc *Compiler, x, y, r []*Wire) error {
	return uintComparator(cc, cc.ZeroWire(), x, y, r)
}

// NewIntGeComparator tests if x>=y.
func NewIntGeComparator(cc *Compiler, x, y, r []*Wire) error {
	return intComparator(cc, cc.OneWire(), x, y, r)
}

// NewUintGeComparator tests if x>=y.
func NewUintGeComparator(cc *Compiler, x, y, r []*Wire) error {
	return uintComparator(cc, cc.OneWire(), x, y, r)
}

// NewIntLtComparator tests if x<y.
func NewIntLtComparator(cc *Compiler, x, y, r []*Wire) error {
	return intComparator(cc, cc.ZeroWire(), y, x, r)
}

// NewUintLtComparator tests if x<y.
func NewUintLtComparator(cc *Compiler, x, y, r []*Wire) error {
	return uintComparator(cc, cc.ZeroWire(), y, x, r)
}

// NewIntLeComparator tests if x<=y.
func NewIntLeComparator(cc *Compiler, x, y, r []*Wire) error {
	return intComparator(cc, cc.OneWire(), y, x, r)
}

// NewUintLeComparator tests if x<=y.
func NewUintLeComparator(cc *Compiler, x, y, r []*Wire) error {
	return uintComparator(cc, cc.OneWire(), y, x, r)
}

// NewNeqComparator tewsts if x!=y.
func NewNeqComparator(cc *Compiler, x, y, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid neq comparator arguments: r=%d", len(r))
	}

	eq := cc.Calloc.Wire()
	err := NewEqComparator(cc, x, y, []*Wire{eq})
	if err != nil {
		return err
	}
	cc.INV(eq, r[0])

	return nil
}

// NewEqComparator tests if x==y.
func NewEqComparator(cc *Compiler, x, y, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid eq comparator arguments: r=%d", len(r))
	}
	x, y = cc.ZeroPad(x, y)

	if len(x) == 1 {
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, x[0], y[0], r[0]))
		return nil
	}
	flags := make([]*Wire, len(x))
	for i := range x {
		flags[i] = cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, x[i], y[i], flags[i]))
	}
	for len(flags) > 2 {
		for i := 0; i < len(flags); i += 2 {
			if i+1 < len(flags) {
				flag := cc.Calloc.Wire()
				cc.AddGate(cc.Calloc.BinaryGate(
					circuit.AND, flags[i], flags[i+1], flag))
				flags[i/2] = flag
			} else {
				flags[i/2] = flags[i]
			}
		}
		flags = flags[:(len(flags)+1)/2]
	}
	cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, flags[0], flags[1], r[0]))
	return nil
}

// NewLogicalAND implements logical AND implementing r=x&y. The input
// and output wires must be 1 bit wide.
func NewLogicalAND(cc *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical and arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, x[0], y[0], r[0]))
	return nil
}

// NewLogicalOR implements logical OR implementing r=x|y.  The input
// and output wires must be 1 bit wide.
func NewLogicalOR(cc *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical or arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	cc.OR(x[0], y[0], r[0])
	return nil
}

// NewBitSetTest tests if the index'th bit of x is set.
func NewBitSetTest(cc *Compiler, x []*Wire, index types.Size, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid bit set test arguments: x=%d, r=%d",
			len(x), len(r))
	}
	if index < types.Size(len(x)) {
		w := cc.ZeroWire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[index], w, r[0]))
	} else {
		r[0] = cc.ZeroWire()
	}
	return nil
}

// NewBitClrTest tests if the index'th bit of x is unset.
func NewBitClrTest(cc *Compiler, x []*Wire, index types.Size, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid bit clear test arguments: x=%d, r=%d",
			len(x), len(r))
	}
	if index < types.Size(len(x)) {
		w := cc.OneWire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[index], w, r[0]))
	} else {
		r[0] = cc.OneWire()
	}
	return nil
}
