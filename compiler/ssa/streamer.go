//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

func (prog *Program) StreamCircuit(conn *p2p.Conn, params *utils.Params,
	inputs *big.Int) ([]*big.Int, error) {

	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Printf(" - Sending program info...\n")
	}
	if err := conn.SendData(key[:]); err != nil {
		return nil, err
	}
	// Our input.
	if err := sendArgument(conn, prog.Inputs[0]); err != nil {
		return nil, err
	}
	// Peer input.
	if err := sendArgument(conn, prog.Inputs[1]); err != nil {
		return nil, err
	}
	// Program outputs.
	if err := conn.SendUint32(len(prog.Outputs)); err != nil {
		return nil, err
	}
	for _, o := range prog.Outputs {
		if err := sendArgument(conn, o); err != nil {
			return nil, err
		}
	}
	// Number of program steps.
	if err := conn.SendUint32(len(prog.Steps)); err != nil {
		return nil, err
	}

	// Collect input wire IDs.
	var ids []circuit.Wire
	for _, w := range prog.InputWires {
		// Program's inputs are unassigned because parser is shared
		// between streaming and non-streaming modes.
		w.ID = prog.nextWireID
		prog.nextWireID++
		ids = append(ids, circuit.Wire(w.ID))
	}

	streaming, err := circuit.NewStreaming(key[:], ids, conn)
	if err != nil {
		return nil, err
	}

	// Select our inputs.
	var n1 []ot.Label
	for i := 0; i < prog.Inputs[0].Size; i++ {
		wire := streaming.GetInput(circuit.Wire(i))

		var n ot.Label
		if inputs.Bit(i) == 1 {
			n = wire.L1
		} else {
			n = wire.L0
		}
		n1 = append(n1, n)
	}

	// Send our inputs.
	for idx, i := range n1 {
		if params.Verbose && false {
			fmt.Printf("N1[%d]:\t%s\n", idx, i)
		}
		if err := conn.SendLabel(i); err != nil {
			return nil, err
		}
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048)
	if err != nil {
		return nil, err
	}

	// Send our public key.
	pub := sender.PublicKey()
	data := pub.N.Bytes()
	if err := conn.SendData(data); err != nil {
		return nil, err
	}
	if err := conn.SendUint32(pub.E); err != nil {
		return nil, err
	}
	conn.Flush()

	// Peer OTs its inputs.
	for i := 0; i < prog.Inputs[1].Size; i++ {
		bit, err := conn.ReceiveUint32()
		if err != nil {
			return nil, err
		}
		wire := streaming.GetInput(circuit.Wire(bit))

		m0Data := wire.L0.Bytes()
		m1Data := wire.L1.Bytes()

		xfer, err := sender.NewTransfer(m0Data, m1Data)
		if err != nil {
			return nil, err
		}

		x0, x1 := xfer.RandomMessages()
		if err := conn.SendData(x0); err != nil {
			return nil, err
		}
		if err := conn.SendData(x1); err != nil {
			return nil, err
		}
		conn.Flush()

		v, err := conn.ReceiveData()
		if err != nil {
			return nil, err
		}
		xfer.ReceiveV(v)

		m0p, m1p, err := xfer.Messages()
		if err != nil {
			return nil, err
		}
		if err := conn.SendData(m0p); err != nil {
			return nil, err
		}
		if err := conn.SendData(m1p); err != nil {
			return nil, err
		}
		conn.Flush()
	}

	// Stream circuit.

	var numGates uint64
	var numNonXOR uint64
	cache := make(map[string]*circuit.Circuit)
	var returnIDs []uint32

	start := time.Now()

	for idx, step := range prog.Steps {
		if idx%100 == 0 {
			elapsed := time.Now().Sub(start)
			done := float64(idx) / float64(len(prog.Steps))
			if done > 0 {
				total := time.Duration(float64(elapsed) / done)
				fmt.Printf("%d/%d\t%s remaining, ready at %s\n",
					idx, len(prog.Steps),
					total-elapsed, start.Add(total).Format(time.Stamp))
			} else {
				fmt.Printf("%d/%d\n", idx, len(prog.Steps))
			}
		}
		instr := step.Instr
		var wires [][]*circuits.Wire
		for _, in := range instr.In {
			w, err := prog.AssignedWires(in.String(), in.Type.Bits)
			if err != nil {
				return nil, err
			}
			wires = append(wires, w)
		}

		var out []*circuits.Wire
		var err error
		if instr.Out != nil {
			out, err = prog.AssignedWires(instr.Out.String(),
				instr.Out.Type.Bits)
			if err != nil {
				return nil, err
			}
		}

		switch instr.Op {

		case Slice:
			if !instr.In[1].Const {
				return nil,
					fmt.Errorf("%s only constant index supported", instr.Op)
			}
			var from int
			switch val := instr.In[1].ConstValue.(type) {
			case int32:
				from = int(val)
			default:
				return nil,
					fmt.Errorf("%s unsupported index type %T", instr.Op, val)
			}

			if !instr.In[2].Const {
				return nil,
					fmt.Errorf("%s only constant index supported", instr.Op)
			}
			var to int
			switch val := instr.In[2].ConstValue.(type) {
			case int32:
				to = int(val)
			default:
				return nil,
					fmt.Errorf("%s unsupported index type %T", instr.Op, val)
			}
			if from >= to {
				return nil, fmt.Errorf("%s bounds out of range [%d:%d]",
					instr.Op, from, to)
			}
			prog.SetWires(instr.Out.String(), wires[0][from:to])

		case Mov:
			for bit := 0; bit < instr.Out.Type.Bits; bit++ {
				if bit < len(wires[0]) {
					out[bit].ID = wires[0][bit].ID
				}
				// XXX need ZeroWire
			}

		case Ret:
			if err := conn.SendUint32(circuit.OP_RETURN); err != nil {
				return nil, err
			}
			for _, arg := range wires {
				for _, w := range arg {
					if err := conn.SendUint32(int(w.ID)); err != nil {
						return nil, err
					}
					returnIDs = append(returnIDs, w.ID)
				}
			}
			if circuit.StreamDebug {
				fmt.Printf("return=%v\n", returnIDs)
			}
			conn.Flush()

		case GC:
			wires, ok := prog.wires[instr.GC]
			if ok {
				delete(prog.wires, instr.GC)
				prog.recycleWires(wires)
			} else {
				fmt.Printf("GC: %s not known\n", instr.GC)
			}

		default:
			f, ok := circuitGenerators[instr.Op]
			if !ok {
				return nil, fmt.Errorf("Program.Stream: %s not implemented yet",
					instr.Op)
			}
			circ, ok := cache[instr.StringTyped()]
			if !ok {
				var cIn [][]*circuits.Wire
				var flat []*circuits.Wire

				for _, in := range instr.In {
					w := circuits.MakeWires(in.Type.Bits)
					cIn = append(cIn, w)
					flat = append(flat, w...)
				}

				cOut := circuits.MakeWires(instr.Out.Type.Bits)
				for i := 0; i < instr.Out.Type.Bits; i++ {
					cOut[i].Output = true
				}

				cc, err := circuits.NewCompiler(params, nil, nil, flat, cOut)
				if err != nil {
					return nil, err
				}
				if params.Verbose {
					fmt.Printf("%05d: %s\n", idx, instr.StringTyped())
				}
				err = f(cc, instr, cIn, cOut)
				if err != nil {
					return nil, err
				}
				pruned := cc.Prune()
				if params.Verbose {
					fmt.Printf("%05d: - pruned %d gates\n",
						idx, pruned)
				}
				circ = cc.Compile()
				cache[instr.StringTyped()] = circ
				if params.Verbose {
					fmt.Printf("%05d: - %s\n", idx, circ)
				}
			}
			if false {
				circ.Dump()
				fmt.Printf("%05d: - circuit: %s\n", idx, circ)
			}

			// Collect input and output IDs
			var iIDs, oIDs []circuit.Wire
			var maxID uint32
			for _, vars := range wires {
				for _, w := range vars {
					iIDs = append(iIDs, circuit.Wire(w.ID))
					if w.ID > maxID {
						maxID = w.ID
					}
				}
			}
			for _, w := range out {
				oIDs = append(oIDs, circuit.Wire(w.ID))
				if w.ID > maxID {
					maxID = w.ID
				}
			}

			if err := conn.SendUint32(circuit.OP_CIRCUIT); err != nil {
				return nil, err
			}
			if err := conn.SendUint32(idx); err != nil {
				return nil, err
			}
			if err := conn.SendUint32(circ.NumGates); err != nil {
				return nil, err
			}
			if err := conn.SendUint32(circ.NumWires); err != nil {
				return nil, err
			}
			if err := conn.SendUint32(int(maxID + 1)); err != nil {
				return nil, err
			}

			gStart := time.Now()
			err := streaming.Garble(circ, iIDs, oIDs)
			if err != nil {
				return nil, err
			}
			dt := time.Now().Sub(gStart)
			elapsed := float64(time.Now().UnixNano() - gStart.UnixNano())
			elapsed /= 1000000000
			if elapsed > 0 && false {
				fmt.Printf("%05d: - garbled %10.0f gates/s (%s)\n",
					idx, float64(circ.NumGates)/elapsed, dt)
			}
			numGates += uint64(circ.NumGates)
			numNonXOR += uint64(circ.Stats[circuit.AND])
			numNonXOR += uint64(circ.Stats[circuit.OR])
			numNonXOR += uint64(circ.Stats[circuit.INV])
		}
	}

	op, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	if op != circuit.OP_RESULT {
		return nil, fmt.Errorf("unexpected operation: %d", op)
	}

	result := new(big.Int)

	for i := 0; i < prog.Outputs.Size(); i++ {
		label, err := conn.ReceiveLabel()
		if err != nil {
			return nil, err
		}
		wire := streaming.GetInput(circuit.Wire(returnIDs[i]))
		var bit uint
		if label.Equal(wire.L0) {
			bit = 0
		} else if label.Equal(wire.L1) {
			bit = 1
		} else {
			return nil, fmt.Errorf("unknown label %s for result %d",
				label, i)
		}
		result.SetBit(result, i, bit)
	}
	data = result.Bytes()
	if err := conn.SendData(data); err != nil {
		return nil, err
	}
	conn.Flush()

	fmt.Printf("Max permanent wires: %d, cached circuits: %d\n",
		prog.nextWireID, len(cache))
	fmt.Printf("#gates=%d, #non-XOR=%d\n", numGates, numNonXOR)

	return prog.Outputs.Split(result), nil
}

func sendArgument(conn *p2p.Conn, arg circuit.IOArg) error {
	if err := conn.SendString(arg.Name); err != nil {
		return err
	}
	if err := conn.SendString(arg.Type); err != nil {
		return err
	}
	if err := conn.SendUint32(arg.Size); err != nil {
		return err
	}

	if err := conn.SendUint32(len(arg.Compound)); err != nil {
		return err
	}
	for _, a := range arg.Compound {
		if err := sendArgument(conn, a); err != nil {
			return err
		}
	}

	return nil
}

type NewCircuit func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error

type NewBinary func(cc *circuits.Compiler, a, b []*circuits.Wire,
	out []*circuits.Wire) error

func newBinary(bin NewBinary) NewCircuit {
	return func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return bin(cc, in[0], in[1], out)
	}
}

func newMultiplier(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewMultiplier(cc, cc.Params.CircMultArrayTreshold,
		in[0], in[1], out)
}

func newDivider(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, in[0], in[1], out, nil)
}

func newModulo(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, in[0], in[1], nil, out)
}

var circuitGenerators = map[Operand]NewCircuit{
	Iadd:  newBinary(circuits.NewAdder),
	Uadd:  newBinary(circuits.NewAdder),
	Isub:  newBinary(circuits.NewSubtractor),
	Usub:  newBinary(circuits.NewSubtractor),
	Imult: newMultiplier,
	Umult: newMultiplier,
	Idiv:  newDivider,
	Udiv:  newDivider,
	Imod:  newModulo,
	Umod:  newModulo,
	Ilt:   newBinary(circuits.NewLtComparator),
	Ult:   newBinary(circuits.NewLtComparator),
	Ile:   newBinary(circuits.NewLeComparator),
	Ule:   newBinary(circuits.NewLeComparator),
	Igt:   newBinary(circuits.NewGtComparator),
	Ugt:   newBinary(circuits.NewGtComparator),
	Ige:   newBinary(circuits.NewGeComparator),
	Uge:   newBinary(circuits.NewGeComparator),
	Eq:    newBinary(circuits.NewEqComparator),
	Neq:   newBinary(circuits.NewNeqComparator),
	And:   newBinary(circuits.NewLogicalAND),
	Or:    newBinary(circuits.NewLogicalOR),
	Band:  newBinary(circuits.NewBinaryAND),
	Bclr:  newBinary(circuits.NewBinaryClear),
	Bor:   newBinary(circuits.NewBinaryOR),
	Bxor:  newBinary(circuits.NewBinaryXOR),

	Builtin: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return instr.Builtin(cc, in[0], in[1], out)
	},
	Phi: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return circuits.NewMUX(cc, in[0], in[1], in[2], out)
	},
	Bts: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		if !instr.In[1].Const {
			return fmt.Errorf("%s only constant index supported", instr.Op)
		}
		var index int
		switch val := instr.In[1].ConstValue.(type) {
		case int32:
			index = int(val)
		default:
			return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
		}
		return circuits.NewBitSetTest(cc, in[0], index, out)
	},
	Btc: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		if !instr.In[1].Const {
			return fmt.Errorf("%s only constant index supported", instr.Op)
		}
		var index int
		switch val := instr.In[1].ConstValue.(type) {
		case int32:
			index = int(val)
		default:
			return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
		}
		return circuits.NewBitClrTest(cc, in[0], index, out)
	},
}
