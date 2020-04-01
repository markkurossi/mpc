//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var reParts = regexp.MustCompilePOSIX("[[:space:]]+")

type Seen []bool

func (s Seen) Set(index int) error {
	if index < 0 || index >= len(s) {
		return fmt.Errorf("invalid wire %d [0...%d[", index, len(s))
	}
	s[index] = true
	return nil
}

func Parse(file string) (*Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if strings.HasSuffix(file, ".circ") || strings.HasSuffix(file, ".bristol") {
		return ParseBristol(f)
	} else if strings.HasSuffix(file, ".mpclc") {
		return ParseMPCLC(f)
	}
	return nil, fmt.Errorf("unsupported circuit format")
}

func ParseMPCLC(in io.Reader) (*Circuit, error) {
	r := bufio.NewReader(in)

	var header struct {
		Magic      uint32
		NumGates   uint32
		NumWires   uint32
		NumInputs  uint32
		NumOutputs uint32
	}
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	var inputs, outputs IO
	var inputWires, outputWires int
	var ui32 uint32

	wiresSeen := make(Seen, header.NumWires)

	for i := 0; i < int(header.NumInputs); i++ {
		if err := binary.Read(r, binary.BigEndian, &ui32); err != nil {
			return nil, err
		}
		inputs = append(inputs, IOArg{
			Name: fmt.Sprintf("NI%d", i),
			Type: fmt.Sprintf("u%d", ui32),
			Size: int(ui32),
		})
		inputWires += int(ui32)
	}
	for i := 0; i < int(header.NumOutputs); i++ {
		if err := binary.Read(r, binary.BigEndian, &ui32); err != nil {
			return nil, err
		}
		outputs = append(outputs, IOArg{
			Name: fmt.Sprintf("NO%d", i),
			Type: fmt.Sprintf("u%d", ui32),
			Size: int(ui32),
		})
		outputWires += int(ui32)
	}

	// Mark input wires seen.
	for i := 0; i < inputWires; i++ {
		if err := wiresSeen.Set(i); err != nil {
			return nil, err
		}
	}

	gates := make([]Gate, header.NumGates)
	stats := make(map[Operation]int)
	var gate int
	for gate = 0; ; gate++ {
		op, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch Operation(op) {
		case XOR, XNOR, AND, OR:
			var bin struct {
				Input0 uint32
				Input1 uint32
				Output uint32
			}
			if err := binary.Read(r, binary.BigEndian, &bin); err != nil {
				return nil, err
			}
			if !wiresSeen[bin.Input0] {
				return nil, fmt.Errorf("input %d of gate %d not set",
					bin.Input0, gate)
			}
			if !wiresSeen[bin.Input1] {
				return nil, fmt.Errorf("input %d of gate %d not set",
					bin.Input1, gate)
			}
			if err := wiresSeen.Set(int(bin.Output)); err != nil {
				return nil, err
			}
			gates[gate] = Gate{
				Input0: Wire(bin.Input0),
				Input1: Wire(bin.Input1),
				Output: Wire(bin.Output),
				Op:     Operation(op),
			}

		case INV:
			var unary struct {
				Input0 uint32
				Output uint32
			}
			if err := binary.Read(r, binary.BigEndian, &unary); err != nil {
				return nil, err
			}
			if !wiresSeen[unary.Input0] {
				return nil, fmt.Errorf("input %d of gate %d not set",
					unary.Input0, gate)
			}
			if err := wiresSeen.Set(int(unary.Output)); err != nil {
				return nil, err
			}
			gates[gate] = Gate{
				Input0: Wire(unary.Input0),
				Output: Wire(unary.Output),
				Op:     Operation(op),
			}

		default:
			return nil, fmt.Errorf("unsupported gate type %s", Operation(op))
		}
		count := stats[Operation(op)]
		count++
		stats[Operation(op)] = count
	}

	if uint32(gate) != header.NumGates {
		return nil, fmt.Errorf("not enough gates: got %d, expected %d",
			gate, header.NumGates)
	}

	// Check that all wires are seen.
	for i := 0; i < len(wiresSeen); i++ {
		if !wiresSeen[i] {
			return nil, fmt.Errorf("wire %d not assigned", i)
		}
	}

	return &Circuit{
		NumGates: int(header.NumGates),
		NumWires: int(header.NumWires),
		Inputs:   inputs,
		Outputs:  outputs,
		Gates:    gates,
		Stats:    stats,
	}, nil
}

func ParseBristol(in io.Reader) (*Circuit, error) {
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

	// Mark input wires seen.
	for i := 0; i < inputWires; i++ {
		if err := wiresSeen.Set(i); err != nil {
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
			return nil, fmt.Errorf("wire %d not assigned", i)
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
