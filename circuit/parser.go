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
	"regexp"
	"strconv"
)

type Operation byte

const (
	XOR Operation = iota
	AND
	INV
)

var reParts = regexp.MustCompilePOSIX("[[:space:]]+")

func (op Operation) String() string {
	switch op {
	case XOR:
		return "XOR"
	case AND:
		return "AND"
	case INV:
		return "INV"
	default:
		return fmt.Sprintf("{Operation %d}", op)
	}
}

type Circuit struct {
	NumGates int
	NumWires int
	N1       int
	N2       int
	N3       int
	Gates    map[int]*Gate
}

func (c *Circuit) String() string {
	return fmt.Sprintf("#gates=%d, #wires=%d n1=%d, n2=%d, n3=%d",
		c.NumGates, c.NumWires, c.N1, c.N2, c.N3)
}

type Gate struct {
	ID      uint32
	Inputs  []Wire
	Outputs []Wire
	Op      Operation
}

type Wire uint32

func (w Wire) ID() int {
	return int(w)
}

func (w Wire) String() string {
	return fmt.Sprintf("w%d", w)
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

	// N1 N2 N3
	line, err = readLine(r)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if len(line) != 3 {
		return nil, errors.New("Invalid 2nd line")
	}
	n1, err := strconv.Atoi(line[0])
	if err != nil {
		return nil, err
	}
	n2, err := strconv.Atoi(line[1])
	if err != nil {
		return nil, err
	}
	n3, err := strconv.Atoi(line[2])
	if err != nil {
		return nil, err
	}

	gates := make(map[int]*Gate)
	for gate := 0; ; gate++ {
		line, err = readLine(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
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
			inputs = append(inputs, Wire(v))
		}

		var outputs []Wire
		for i := 0; i < n2; i++ {
			v, err := strconv.Atoi(line[2+n1+i])
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, Wire(v))
		}
		var op Operation
		switch line[len(line)-1] {
		case "XOR":
			op = XOR
		case "AND":
			op = AND
		case "INV":
			op = INV
		default:
			return nil, fmt.Errorf("Invalid operation '%s'", line[len(line)-1])
		}

		gates[gate] = &Gate{
			ID:      uint32(gate),
			Inputs:  inputs,
			Outputs: outputs,
			Op:      op,
		}
	}

	return &Circuit{
		NumGates: numGates,
		NumWires: numWires,
		N1:       n1,
		N2:       n2,
		N3:       n3,
		Gates:    gates,
	}, nil
}

func readLine(r *bufio.Reader) ([]string, error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if len(line) == 1 {
			continue
		}
		parts := reParts.Split(line[:len(line)-1], -1)
		if len(parts) > 0 {
			return parts, nil
		}
	}
}
