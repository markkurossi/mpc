//
// Copyright (c) 2019-2022 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/markkurossi/tabulate"
)

// Operation specifies gate function.
type Operation byte

// Gate functions.
const (
	XOR Operation = iota
	XNOR
	AND
	OR
	INV
	Count
	NumLevels
	MaxWidth
)

// Stats holds statistics about circuit operations.
type Stats [MaxWidth + 1]uint64

// Add adds the argument statistics to this statistics object.
func (stats *Stats) Add(o Stats) {
	for i := XOR; i < Count; i++ {
		stats[i] += o[i]
	}
	stats[Count]++

	for i := NumLevels; i <= MaxWidth; i++ {
		if o[i] > stats[i] {
			stats[i] = o[i]
		}
	}
}

// Count returns the number of gates in the statistics object.
func (stats Stats) Count() uint64 {
	var result uint64
	for i := XOR; i < Count; i++ {
		result += stats[i]
	}
	return result
}

// Cost computes the relative computational cost of the circuit.
func (stats Stats) Cost() uint64 {
	return (stats[AND]+stats[INV])*2 + stats[OR]*3
}

func (stats Stats) String() string {
	var result string

	for i := XOR; i < Count; i++ {
		v := stats[i]
		if len(result) > 0 {
			result += " "
		}
		result += fmt.Sprintf("%s=%d", i, v)
	}
	result += fmt.Sprintf(" xor=%d", stats[XOR]+stats[XNOR])
	result += fmt.Sprintf(" !xor=%d", stats[AND]+stats[OR]+stats[INV])
	result += fmt.Sprintf(" levels=%d", stats[NumLevels])
	result += fmt.Sprintf(" width=%d", stats[MaxWidth])
	return result
}

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
	case Count:
		return "#"
	default:
		return fmt.Sprintf("{Operation %d}", op)
	}
}

// IOArg describes circuit input argument.
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

// Parse parses the I/O argument from the input string values.
func (io IOArg) Parse(inputs []string) (*big.Int, error) {
	result := new(big.Int)

	if len(io.Compound) == 0 {
		if len(inputs) != 1 {
			return nil,
				fmt.Errorf("invalid amount of arguments, got %d, expected 1",
					len(inputs))
		}

		if strings.HasPrefix(io.Type, "u") || strings.HasPrefix(io.Type, "i") {
			_, ok := result.SetString(inputs[0], 0)
			if !ok {
				return nil, fmt.Errorf("invalid input: %s", inputs[0])
			}
		} else if io.Type == "bool" {
			switch inputs[0] {
			case "0", "f", "false":
			case "1", "t", "true":
				result.SetInt64(1)
			default:
				return nil, fmt.Errorf("invalid bool constant: %s", inputs[0])
			}
		} else {
			ok, count, elSize, _ := ParseArrayType(io.Type)
			if ok {
				val := new(big.Int)
				_, ok := val.SetString(inputs[0], 0)
				if !ok {
					return nil, fmt.Errorf("invalid input: %s", inputs[0])
				}

				valElCount := val.BitLen() / elSize
				if val.BitLen()%elSize != 0 {
					valElCount++
				}
				if valElCount > count {
					return nil, fmt.Errorf("too many values for input: %s",
						inputs[0])
				}
				pad := count - valElCount
				val.Lsh(val, uint(pad*elSize))

				mask := new(big.Int)
				for i := 0; i < elSize; i++ {
					mask.SetBit(mask, i, 1)
				}

				for i := 0; i < count; i++ {
					next := new(big.Int).Rsh(val, uint((count-i-1)*elSize))
					next = next.And(next, mask)

					next.Lsh(next, uint(i*elSize))
					result.Or(result, next)
				}
			} else {
				return nil, fmt.Errorf("unsupported input type: %s", io.Type)
			}
		}

		return result, nil
	}
	if len(inputs) != len(io.Compound) {
		return nil,
			fmt.Errorf("invalid amount of arguments, got %d, expected %d",
				len(inputs), len(io.Compound))
	}

	var offset int

	for idx, arg := range io.Compound {
		input, err := arg.Parse(inputs[idx : idx+1])
		if err != nil {
			return nil, err
		}

		input.Lsh(input, uint(offset))
		result.Or(result, input)

		offset += arg.Size
	}
	return result, nil
}

var reArr = regexp.MustCompilePOSIX(`^\[([[:digit:]]+)\](.+)$`)
var reSized = regexp.MustCompilePOSIX(`^[[:^digit:]]+([[:digit:]]+)$`)

// ParseArrayType parses the argument value as array type.
func ParseArrayType(val string) (ok bool, count, elementSize int,
	elementType string) {

	matches := reArr.FindStringSubmatch(val)
	if matches == nil {
		return
	}
	var err error
	count, err = strconv.Atoi(matches[1])
	if err != nil {
		panic(fmt.Sprintf("invalid array size: %s", matches[1]))
	}
	ok = true
	elementSize = Size(matches[2])
	elementType = matches[2]
	return
}

// Size returns the type size in bits.
func Size(t string) int {
	matches := reArr.FindStringSubmatch(t)
	if matches != nil {
		count, err := strconv.Atoi(matches[1])
		if err != nil {
			panic(fmt.Sprintf("invalid array size: %s", matches[1]))
		}
		return count * Size(matches[2])
	}
	matches = reSized.FindStringSubmatch(t)
	if matches == nil {
		panic(fmt.Sprintf("invalid type: %s", t))
	}
	bits, err := strconv.Atoi(matches[1])
	if err != nil {
		panic(fmt.Sprintf("invalid bit count: %s", matches[1]))
	}
	return bits
}

// IO specifies circuit input and output arguments.
type IO []IOArg

// Size computes the size of the circuit input and output arguments in
// bits.
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

// Split splits the value into separate I/O arguments.
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

// Circuit specifies a boolean circuit.
type Circuit struct {
	NumGates int
	NumWires int
	Inputs   IO
	Outputs  IO
	Gates    []Gate
	Stats    Stats
}

func (c *Circuit) String() string {
	return fmt.Sprintf("#gates=%d (%s) #w=%d", c.NumGates, c.Stats, c.NumWires)
}

// TabulateStats prints the circuit stats as a table to the specified
// output Writer.
func (c *Circuit) TabulateStats(out io.Writer) {
	tab := tabulate.New(tabulate.UnicodeLight)
	tab.Header("XOR").SetAlign(tabulate.MR)
	tab.Header("XNOR").SetAlign(tabulate.MR)
	tab.Header("AND").SetAlign(tabulate.MR)
	tab.Header("OR").SetAlign(tabulate.MR)
	tab.Header("INV").SetAlign(tabulate.MR)
	tab.Header("Gates").SetAlign(tabulate.MR)
	tab.Header("XOR").SetAlign(tabulate.MR)
	tab.Header("!XOR").SetAlign(tabulate.MR)
	tab.Header("Wires").SetAlign(tabulate.MR)

	c.TabulateRow(tab.Row())
	tab.Print(out)
}

// TabulateRow tabulates circuit statistics to the argument tabulation
// row.
func (c *Circuit) TabulateRow(row *tabulate.Row) {
	var sumGates uint64
	for op := XOR; op < Count; op++ {
		row.Column(fmt.Sprintf("%v", c.Stats[op]))
		sumGates += c.Stats[op]
	}
	row.Column(fmt.Sprintf("%v", sumGates))
	row.Column(fmt.Sprintf("%v", c.Stats[XOR]+c.Stats[XNOR]))
	row.Column(fmt.Sprintf("%v", c.Stats[AND]+c.Stats[OR]+c.Stats[INV]))
	row.Column(fmt.Sprintf("%v", c.NumWires))
}

// Cost computes the relative computational cost of the circuit.
func (c *Circuit) Cost() uint64 {
	return c.Stats.Cost()
}

// Dump prints a debug dump of the circuit.
func (c *Circuit) Dump() {
	fmt.Printf("circuit %s\n", c)
	for id, gate := range c.Gates {
		fmt.Printf("%04d\t%s\n", id, gate)
	}
}

// AssignLevels assigns levels for gates. The level desribes how many
// steps away the gate is from input wires.
func (c *Circuit) AssignLevels() {
	levels := make([]Level, c.NumWires)
	countByLevel := make([]uint32, c.NumWires)

	var max Level

	for idx, gate := range c.Gates {
		level := levels[gate.Input0]
		if gate.Op != INV {
			l1 := levels[gate.Input1]
			if l1 > level {
				level = l1
			}
		}
		c.Gates[idx].Level = level
		countByLevel[level]++

		level++

		levels[gate.Output] = level
		if level > max {
			max = level
		}
	}
	c.Stats[NumLevels] = uint64(max)

	var maxWidth uint32
	for _, count := range countByLevel {
		if count > maxWidth {
			maxWidth = count
		}
	}
	if false {
		for i := 0; i < int(max); i++ {
			fmt.Printf("%v,%v\n", i, countByLevel[i])
		}
	}

	c.Stats[MaxWidth] = uint64(maxWidth)
}

// Level defines gate's distance from input wires.
type Level uint32

// Gate specifies a boolean gate.
type Gate struct {
	Input0 Wire
	Input1 Wire
	Output Wire
	Op     Operation
	Level  Level
}

func (g Gate) String() string {
	return fmt.Sprintf("%v %v %v", g.Inputs(), g.Op, g.Output)
}

// Inputs returns gate input wires.
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

// Wire specifies a wire ID.
type Wire uint32

// InvalidWire specifies an invalid wire ID.
const InvalidWire Wire = math.MaxUint32

// ID returns the wire ID as integer.
func (w Wire) ID() int {
	return int(w)
}

func (w Wire) String() string {
	return fmt.Sprintf("w%d", w)
}
