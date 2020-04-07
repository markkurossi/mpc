//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"math/big"
)

type Operation byte

const (
	XOR Operation = iota
	XNOR
	AND
	OR
	INV
)

func (op Operation) String() string {
	switch op {
	case XOR:
		return "XOR"
	case XNOR:
		return "XNOR"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case INV:
		return "INV"
	default:
		return fmt.Sprintf("{Operation %d}", op)
	}
}

type IOArg struct {
	Name     string
	Type     string
	Size     int
	Compound IO
}

func (io IOArg) String() string {
	if len(io.Compound) > 0 {
		return io.Compound.String()
	}

	if len(io.Name) > 0 {
		return io.Name + ":" + io.Type
	}
	return io.Type
}

func (io IOArg) Parse(inputs []string) (*big.Int, error) {
	if len(io.Compound) == 0 {
		if len(inputs) != 1 {
			return nil,
				fmt.Errorf("invalid amount of arguments, got %d, expected 1",
					len(inputs))
		}
		i := new(big.Int)
		_, ok := i.SetString(inputs[0], 0)
		if !ok {
			return nil, fmt.Errorf("invalid input: %s", inputs[0])
		}
		return i, nil
	}
	if len(inputs) != len(io.Compound) {
		return nil,
			fmt.Errorf("invalid amount of arguments, got %d, expected %d",
				len(inputs), len(io.Compound))
	}

	result := new(big.Int)
	var offset int

	for idx, arg := range io.Compound {
		i := new(big.Int)
		// XXX Type checks
		_, ok := i.SetString(inputs[idx], 0)
		if !ok {
			return nil, fmt.Errorf("invalid input: %s", inputs[idx])
		}
		i.Lsh(i, uint(offset))
		result.Or(result, i)

		offset += arg.Size
	}
	return result, nil
}

type IO []IOArg

func (io IO) Size() int {
	var sum int
	for _, a := range io {
		sum += a.Size
	}
	return sum
}

func (io IO) String() string {
	var str = ""
	for i, a := range io {
		if i > 0 {
			str += ", "
		}
		if len(a.Name) > 0 {
			str += a.Name + ":"
		}
		str += a.Type
	}
	return str
}

func (io IO) Split(in *big.Int) []*big.Int {
	var result []*big.Int
	var bit int
	for _, arg := range io {
		r := big.NewInt(0)
		for i := 0; i < arg.Size; i++ {
			if in.Bit(bit) == 1 {
				r = big.NewInt(0).SetBit(r, i, 1)
			}
			bit++
		}
		result = append(result, r)
	}
	return result
}

type Circuit struct {
	NumGates int
	NumWires int
	Inputs   IO
	Outputs  IO
	Gates    []Gate
	Stats    map[Operation]int
}

func (c *Circuit) String() string {
	var stats string

	for k := XOR; k <= INV; k++ {
		v := c.Stats[k]
		if len(stats) > 0 {
			stats += " "
		}
		stats += fmt.Sprintf("%s=%d", k, v)
	}
	return fmt.Sprintf("#gates=%d (%s) #w=%d", c.NumGates, stats, c.NumWires)
}

func (c *Circuit) Cost() int {
	return (c.Stats[AND]+c.Stats[OR])*4 + c.Stats[INV]*2
}

func (c *Circuit) Dump() {
	fmt.Printf("circuit %s\n", c)
	for id, gate := range c.Gates {
		fmt.Printf("%04d\t%s\n", id, gate)
	}
}

type Gate struct {
	Input0 Wire
	Input1 Wire
	Output Wire
	Op     Operation
}

func (g Gate) String() string {
	return fmt.Sprintf("%v %v %v", g.Inputs(), g.Op, g.Output)
}

func (g Gate) Inputs() []Wire {
	switch g.Op {
	case XOR, XNOR, AND, OR:
		return []Wire{g.Input0, g.Input1}
	case INV:
		return []Wire{g.Input0}
	default:
		panic(fmt.Sprintf("unsupported gate type %s", g.Op))
	}
}

type Wire uint32

const (
	TmpWireID = 0x80000000
)

func (w Wire) ID() int {
	return int(w)
}

func (w Wire) String() string {
	if w >= TmpWireID {
		return fmt.Sprintf("~%d", w&^TmpWireID)
	} else {
		return fmt.Sprintf("w%d", w)
	}
}
