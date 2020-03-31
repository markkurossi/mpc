//
// parser.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

type Operation byte

const (
	XOR Operation = iota
	XNOR
	AND
	OR
	INV
)

var reParts = regexp.MustCompilePOSIX("[[:space:]]+")

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
	Combound IO
}

func (io IOArg) String() string {
	if len(io.Combound) > 0 {
		return io.Combound.String()
	}

	if len(io.Name) > 0 {
		return io.Name + ":" + io.Type
	}
	return io.Type
}

func (io IOArg) Parse(inputs []string) (*big.Int, error) {
	if len(io.Combound) == 0 {
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
	if len(inputs) != len(io.Combound) {
		return nil,
			fmt.Errorf("invalid amount of arguments, got %d, expected %d",
				len(inputs), len(io.Combound))
	}

	result := new(big.Int)
	var offset int

	for idx, arg := range io.Combound {
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
	return fmt.Sprintf("#gates=%d (%s)", c.NumGates, stats)
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

func (c *Circuit) Marshal(out io.Writer) {
	fmt.Fprintf(out, "%d %d\n", c.NumGates, c.NumWires)
	fmt.Fprintf(out, "%d", len(c.Inputs))
	for _, input := range c.Inputs {
		fmt.Fprintf(out, " %d", input.Size)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d", len(c.Outputs))
	for _, ret := range c.Outputs {
		fmt.Fprintf(out, " %d", ret.Size)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)

	for _, g := range c.Gates {
		fmt.Fprintf(out, "%d 1", len(g.Inputs()))
		for _, w := range g.Inputs() {
			fmt.Fprintf(out, " %d", w)
		}
		fmt.Fprintf(out, " %d", g.Output)
		fmt.Fprintf(out, " %s\n", g.Op)
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

func (w Wire) ID() int {
	return int(w)
}

func (w Wire) String() string {
	return fmt.Sprintf("w%d", w)
}

type Seen []bool

func (s Seen) Set(index int) error {
	if index < 0 || index >= len(s) {
		return fmt.Errorf("invalid wire %d [0...%d[", index, len(s))
	}
	s[index] = true
	return nil
}

func Parse(in io.Reader) (*Circuit, error) {
	r := bufio.NewReader(in)

	// NumGates NumWires
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	if len(line) != 2 {
		fmt.Printf("Line: %v\n", line)
		return nil, errors.New("Invalid 1st line")
	}
	numGates, err := strconv.Atoi(line[0])
	if err != nil {
		return nil, err
	}
	numWires, err := strconv.Atoi(line[1])
	if err != nil {
		return nil, err
	}
	wiresSeen := make(Seen, numWires)

	// Inputs
	line, err = readLine(r)
	if err != nil {
		return nil, err
	}
	niv, err := strconv.Atoi(line[0])
	if err != nil {
		return nil, err
	}
	if 1+niv != len(line) {
		return nil, fmt.Errorf("invalid inputs line: niv=%d, len=%d",
			niv, len(line))
	}
	var inputs IO
	var inputWires int
	for i := 1; i < len(line); i++ {
		bits, err := strconv.Atoi(line[i])
		if err != nil {
			return nil, fmt.Errorf("invalid input bits: %s", err)
		}
		inputs = append(inputs, IOArg{
			Name: fmt.Sprintf("NI%d", i),
			Type: fmt.Sprintf("u%d", bits),
			Size: bits,
		})
		inputWires += bits
	}
	if inputWires == 0 {
		return nil, fmt.Errorf("no inputs defined")
	}

	// Mark input wires set.
	for i := 0; i < inputWires; i++ {
		err = wiresSeen.Set(i)
		if err != nil {
			return nil, err
		}
	}

	// Outputs
	line, err = readLine(r)
	if err != nil {
		return nil, err
	}
	nov, err := strconv.Atoi(line[0])
	if err != nil {
		return nil, err
	}
	if 1+nov != len(line) {
		return nil, errors.New("invalid outputs line")
	}
	var outputs IO
	for i := 1; i < len(line); i++ {
		bits, err := strconv.Atoi(line[i])
		if err != nil {
			return nil, fmt.Errorf("invalid output bits: %s", err)
		}
		outputs = append(outputs, IOArg{
			Name: fmt.Sprintf("NO%d", i),
			Type: fmt.Sprintf("u%d", bits),
			Size: bits,
		})
	}

	gates := make([]Gate, numGates)
	stats := make(map[Operation]int)
	var gate int
	for gate = 0; ; gate++ {
		line, err = readLine(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if gate >= numGates {
			return nil, errors.New("too many gates")
		}
		if len(line) < 3 {
			return nil, fmt.Errorf("Invalid gate: %v", line)
		}
		n1, err := strconv.Atoi(line[0])
		if err != nil {
			return nil, err
		}
		n2, err := strconv.Atoi(line[1])
		if err != nil {
			return nil, err
		}
		if 2+n1+n2+1 != len(line) {
			return nil, fmt.Errorf("Invalid gate: %v", line)
		}

		var inputs []Wire
		for i := 0; i < n1; i++ {
			v, err := strconv.Atoi(line[2+i])
			if err != nil {
				return nil, err
			}
			if !wiresSeen[v] {
				return nil, fmt.Errorf("input %d of gate %d not set", v, gate)
			}
			inputs = append(inputs, Wire(v))
		}

		var outputs []Wire
		for i := 0; i < n2; i++ {
			v, err := strconv.Atoi(line[2+n1+i])
			if err != nil {
				return nil, err
			}
			err = wiresSeen.Set(v)
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, Wire(v))
		}
		var op Operation
		var numInputs int
		switch line[len(line)-1] {
		case "XOR":
			op = XOR
			numInputs = 2
		case "XNOR":
			op = XNOR
			numInputs = 2
		case "AND":
			op = AND
			numInputs = 2
		case "OR":
			op = OR
			numInputs = 2
		case "INV":
			op = INV
			numInputs = 1
		default:
			return nil, fmt.Errorf("Invalid operation '%s'", line[len(line)-1])
		}

		if len(inputs) != numInputs {
			return nil, fmt.Errorf("invalid number of inputs %d for %s",
				len(inputs), op)
		}
		if len(outputs) != 1 {
			return nil, fmt.Errorf("invalid number of outputs %d for %s",
				len(outputs), op)
		}

		var input1 Wire
		if len(inputs) > 1 {
			input1 = inputs[1]
		}

		gates[gate] = Gate{
			Input0: inputs[0],
			Input1: input1,
			Output: outputs[0],
			Op:     op,
		}
		count := stats[op]
		count++
		stats[op] = count
	}
	if gate != numGates {
		return nil, fmt.Errorf("not enough gates: got %d, expected %d",
			gate, numGates)
	}

	// Check that all wires are seen.
	for i := 0; i < len(wiresSeen); i++ {
		if !wiresSeen[i] {
			return nil, fmt.Errorf("wire %d not assigned\n", i)
		}
	}

	return &Circuit{
		NumGates: numGates,
		NumWires: numWires,
		Inputs:   inputs,
		Outputs:  outputs,
		Gates:    gates,
		Stats:    stats,
	}, nil
}

func readLine(r *bufio.Reader) ([]string, error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		parts := reParts.Split(line, -1)
		if len(parts) > 0 {
			return parts, nil
		}
	}
}
