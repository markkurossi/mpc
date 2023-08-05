//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"sort"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
	"github.com/markkurossi/mpc/types"
	"github.com/markkurossi/tabulate"
)

// StreamCircuit streams the program circuit into the P2P connection.
func (prog *Program) StreamCircuit(conn *p2p.Conn, oti ot.OT,
	params *utils.Params, inputs *big.Int, timing *circuit.Timing) (
	circuit.IO, []*big.Int, error) {

	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, nil, err
	}

	if params.Verbose {
		fmt.Printf(" - Sending program info...\n")
	}
	if err := conn.SendData(key[:]); err != nil {
		return nil, nil, err
	}
	// Our input.
	if err := sendArgument(conn, prog.Inputs[0]); err != nil {
		return nil, nil, err
	}
	// Peer input.
	if err := sendArgument(conn, prog.Inputs[1]); err != nil {
		return nil, nil, err
	}
	// Program outputs.
	if err := conn.SendUint32(len(prog.Outputs)); err != nil {
		return nil, nil, err
	}
	for _, o := range prog.Outputs {
		if err := sendArgument(conn, o); err != nil {
			return nil, nil, err
		}
	}
	// Number of program steps.
	if err := conn.SendUint32(len(prog.Steps)); err != nil {
		return nil, nil, err
	}

	// Collect input wire IDs.
	var ids []circuit.Wire
	for _, w := range prog.InputWires {
		// Program's inputs are unassigned because parser is shared
		// between streaming and non-streaming modes.
		w.SetID(prog.nextWireID)
		prog.nextWireID++
		ids = append(ids, circuit.Wire(w.ID()))
	}

	streaming, err := circuit.NewStreaming(key[:], ids, conn)
	if err != nil {
		return nil, nil, err
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
	var labelData ot.LabelData
	for idx, i := range n1 {
		if params.Verbose && false {
			fmt.Printf("N1[%d]:\t%s\n", idx, i)
		}
		if err := conn.SendLabel(i, &labelData); err != nil {
			return nil, nil, err
		}
	}

	ioStats := conn.Stats.Sum()
	timing.Sample("Init", []string{circuit.FileSize(ioStats).String()})

	// Init oblivious transfer.
	err = oti.InitSender(conn)
	if err != nil {
		return nil, nil, err
	}
	xfer := conn.Stats.Sum() - ioStats
	ioStats = conn.Stats.Sum()
	timing.Sample("OT Init", []string{circuit.FileSize(xfer).String()})

	// Peer OTs its inputs.
	err = oti.Send(streaming.GetInputs(prog.Inputs[0].Size,
		prog.Inputs[1].Size))
	if err != nil {
		return nil, nil, err
	}
	xfer = conn.Stats.Sum() - ioStats
	ioStats = conn.Stats.Sum()
	timing.Sample("Peer Inputs", []string{circuit.FileSize(xfer).String()})

	zero, err := prog.ZeroWire(conn, streaming)
	if err != nil {
		return nil, nil, err
	}
	one, err := prog.OneWire(conn, streaming)
	if err != nil {
		return nil, nil, err
	}

	err = prog.DefineConstants(zero, one)
	if err != nil {
		return nil, nil, err
	}

	// Stream circuit.

	cache := make(map[string]*circuit.Circuit)
	var returnIDs []uint32

	start := time.Now()
	lastReport := start
	var circCompileDuration time.Duration

	istats := make(map[string]circuit.Stats)

	var wires [][]*circuits.Wire
	var iIDs, oIDs []circuit.Wire

	for idx, step := range prog.Steps {
		if idx%10 == 0 && params.Verbose {
			now := time.Now()
			if now.Sub(lastReport) > time.Second*5 {
				lastReport = now
				elapsed := now.Sub(start)
				done := float64(idx) / float64(len(prog.Steps))
				if done > 0 {
					total := time.Duration(float64(elapsed) / done)
					progress := fmt.Sprintf("%d/%d", idx, len(prog.Steps))
					remaining := fmt.Sprintf("%24s", total-elapsed)
					fmt.Printf("%-14s\t%s remaining\tETA %s\n",
						progress, remaining,
						start.Add(total).Format(time.Stamp))
				} else {
					fmt.Printf("%d/%d\n", idx, len(prog.Steps))
				}
			}
		}
		instr := step.Instr
		wires = wires[:0]
		for _, in := range instr.In {
			w, err := prog.AssignedWires(in.String(), in.Type.Bits)
			if err != nil {
				return nil, nil, err
			}
			wires = append(wires, w)
		}

		var out []*circuits.Wire
		var err error
		if instr.Out != nil {
			out, err = prog.AssignedWires(instr.Out.String(),
				instr.Out.Type.Bits)
			if err != nil {
				return nil, nil, err
			}
		}

		if params.Verbose && circuit.StreamDebug {
			fmt.Printf("%05d: %s\n", idx, instr.String())
		}

		switch instr.Op {

		case Lshift:
			count, err := instr.In[1].ConstInt()
			if err != nil {
				return nil, nil,
					fmt.Errorf("%s: unsupported index type %T: %s",
						instr.Op, instr.In[1], err)
			}
			if count < 0 {
				return nil, nil,
					fmt.Errorf("%s: negative shift count %d", instr.Op, count)
			}
			for bit := 0; bit < len(out); bit++ {
				var id uint32
				if bit-int(count) >= 0 && bit-int(count) < len(wires[0]) {
					id = wires[0][bit-int(count)].ID()
				} else {
					w, err := prog.ZeroWire(conn, streaming)
					if err != nil {
						return nil, nil, err
					}
					id = w.ID()
				}
				out[bit].SetID(id)
			}

		case Rshift, Srshift:
			var signWire *circuits.Wire
			if instr.Op == Srshift {
				signWire = wires[0][len(wires[0])-1]
			} else {
				zero, err := prog.ZeroWire(conn, streaming)
				if err != nil {
					return nil, nil, err
				}
				signWire = zero
			}
			count, err := instr.In[1].ConstInt()
			if err != nil {
				return nil, nil,
					fmt.Errorf("%s: unsupported index type %T: %s",
						instr.Op, instr.In[1], err)
			}
			if count < 0 {
				return nil, nil,
					fmt.Errorf("%s: negative shift count %d", instr.Op, count)
			}
			for bit := 0; bit < len(out); bit++ {
				var id uint32
				if bit+int(count) < len(wires[0]) {
					id = wires[0][bit+int(count)].ID()
				} else {
					id = signWire.ID()
				}
				out[bit].SetID(id)
			}

		case Slice:
			from, err := instr.In[1].ConstInt()
			if err != nil {
				return nil, nil,
					fmt.Errorf("%s: unsupported index type %T: %s",
						instr.Op, instr.In[1], err)
			}
			to, err := instr.In[2].ConstInt()
			if err != nil {
				return nil, nil,
					fmt.Errorf("%s: unsupported index type %T: %s",
						instr.Op, instr.In[2], err)
			}
			if from >= to {
				return nil, nil, fmt.Errorf("%s: bounds out of range [%d:%d]",
					instr.Op, from, to)
			}
			for bit := from; bit < to; bit++ {
				var id uint32
				if int(bit) < len(wires[0]) {
					id = wires[0][bit].ID()
				} else {
					w, err := prog.ZeroWire(conn, streaming)
					if err != nil {
						return nil, nil, err
					}
					id = w.ID()
				}
				out[bit-from].SetID(id)
			}

		case Mov, Smov:
			var signWire *circuits.Wire
			if instr.Op == Smov {
				signWire = wires[0][len(wires[0])-1]
			} else {
				zero, err := prog.ZeroWire(conn, streaming)
				if err != nil {
					return nil, nil, err
				}
				signWire = zero
			}
			for bit := types.Size(0); bit < instr.Out.Type.Bits; bit++ {
				var id uint32
				if bit < types.Size(len(wires[0])) {
					id = wires[0][bit].ID()
				} else {
					id = signWire.ID()
				}
				out[bit].SetID(id)
			}

		case Amov:
			// v arr from to:
			// array[from:to] = v
			from, err := instr.In[2].ConstInt()
			if err != nil {
				return nil, nil, fmt.Errorf("%s: unsupported index type %T: %s",
					instr.Op, instr.In[2], err)
			}
			to, err := instr.In[3].ConstInt()
			if err != nil {
				return nil, nil, fmt.Errorf("%s: unsupported index type %T: %s",
					instr.Op, instr.In[3], err)
			}
			if from < 0 || from >= to {
				return nil, nil, fmt.Errorf("%s: bounds out of range [%d:%d]",
					instr.Op, from, to)
			}

			for bit := types.Size(0); bit < instr.Out.Type.Bits; bit++ {
				var id uint32
				if bit < from || bit >= to {
					if bit < types.Size(len(wires[1])) {
						id = wires[1][bit].ID()
					} else {
						w, err := prog.ZeroWire(conn, streaming)
						if err != nil {
							return nil, nil, err
						}
						id = w.ID()
					}
				} else {
					idx := bit - from
					if idx < types.Size(len(wires[0])) {
						id = wires[0][idx].ID()
					} else {
						w, err := prog.ZeroWire(conn, streaming)
						if err != nil {
							return nil, nil, err
						}
						id = w.ID()
					}
				}
				out[bit].SetID(id)
			}

		case Ret:
			if err := conn.SendUint32(circuit.OpReturn); err != nil {
				return nil, nil, err
			}
			for _, arg := range wires {
				for _, w := range arg {
					if err := conn.SendUint32(int(w.ID())); err != nil {
						return nil, nil, err
					}
					returnIDs = append(returnIDs, w.ID())
				}
			}
			if circuit.StreamDebug {
				fmt.Printf("return=%v\n", returnIDs)
			}
			if err := conn.Flush(); err != nil {
				return nil, nil, err
			}

		case Circ:
			// Collect input and output IDs
			iIDs = iIDs[:0]
			oIDs = oIDs[:0]
			for i := 0; i < len(wires); i++ {
				for j := 0; j < instr.Circ.Inputs[i].Size; j++ {
					if j < len(wires[i]) {
						iIDs = append(iIDs, circuit.Wire(wires[i][j].ID()))
					} else {
						iIDs = append(iIDs, circuit.Wire(prog.zeroWire.ID()))
					}
				}
			}
			// Return wires.
			for i, ret := range instr.Ret {
				wires, err := prog.AssignedWires(ret.String(), ret.Type.Bits)
				if err != nil {
					return nil, nil, err
				}
				for j := 0; j < instr.Circ.Outputs[i].Size; j++ {
					if j < len(wires) {
						oIDs = append(oIDs, circuit.Wire(wires[j].ID()))
					} else {
						oIDs = append(oIDs, circuit.Wire(prog.zeroWire.ID()))
					}
				}
			}
			if len(oIDs) != instr.Circ.Outputs.Size() {
				return nil, nil, fmt.Errorf("%s: output mismatch: %d vs. %d",
					instr.Op, len(oIDs), instr.Circ.Outputs.Size())
			}
			if params.Verbose && circuit.StreamDebug {
				fmt.Printf("%05d: - circuit: %s\n", idx, instr.Circ)
			}
			if params.Diagnostics {
				addStats(istats, instr, instr.Circ)
			}
			err = prog.garble(conn, streaming, idx, instr.Circ, iIDs, oIDs)
			if err != nil {
				return nil, nil, err
			}

		case GC:
			alloc, ok := prog.wires[instr.GC]
			if ok {
				delete(prog.wires, instr.GC)
				prog.recycleWires(alloc)
			} else {
				fmt.Printf("GC: %s not known\n", instr.GC)
			}

		default:
			f, ok := circuitGenerators[instr.Op]
			if !ok {
				return nil, nil,
					fmt.Errorf("Program.Stream: %s not implemented yet",
						instr.Op)
			}
			if params.Verbose && circuit.StreamDebug {
				fmt.Printf(" - %s\n", instr.StringTyped())
			}
			circ, ok := cache[instr.StringTyped()]
			if !ok {
				var cIn [][]*circuits.Wire
				var flat []*circuits.Wire
				startTime := time.Now()

				for _, in := range wires {
					w := circuits.MakeWires(types.Size(len(in)))
					cIn = append(cIn, w)
					flat = append(flat, w...)
				}

				cOut := circuits.MakeWires(instr.Out.Type.Bits)
				for i := types.Size(0); i < instr.Out.Type.Bits; i++ {
					cOut[i].SetOutput(true)
				}

				cc, err := circuits.NewCompiler(params, nil, nil, flat, cOut)
				if err != nil {
					return nil, nil, err
				}
				cacheable, err := f(cc, instr, cIn, cOut)
				if err != nil {
					return nil, nil, err
				}
				cc.ConstPropagate()
				pruned := cc.Prune()
				if params.Verbose && circuit.StreamDebug {
					fmt.Printf("%05d: - pruned %d gates\n",
						idx, pruned)
				}
				circ = cc.Compile()
				if cacheable {
					cache[instr.StringTyped()] = circ
				}
				if params.Verbose && circuit.StreamDebug {
					fmt.Printf("%05d: - %s\n", idx, circ)
				}
				circ.AssignLevels()
				circCompileDuration += time.Now().Sub(startTime)
			}
			if false {
				circ.Dump()
				fmt.Printf("%05d: - circuit: %s\n", idx, circ)
			}
			if params.Diagnostics {
				addStats(istats, instr, circ)
			}

			// Collect input and output IDs
			iIDs = iIDs[:0]
			oIDs = oIDs[:0]
			for _, vars := range wires {
				for _, w := range vars {
					iIDs = append(iIDs, circuit.Wire(w.ID()))
				}
			}
			for _, w := range out {
				oIDs = append(oIDs, circuit.Wire(w.ID()))
			}

			err = prog.garble(conn, streaming, idx, circ, iIDs, oIDs)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	xfer = conn.Stats.Sum() - ioStats
	ioStats = conn.Stats.Sum()
	sample := timing.Sample("Stream", []string{circuit.FileSize(xfer).String()})
	sample.Samples = append(sample.Samples, &circuit.Sample{
		Label: "Compile",
		Abs:   circCompileDuration,
	})
	sample.Samples = append(sample.Samples, &circuit.Sample{
		Label: "Garble",
		Abs:   prog.garbleDuration,
	})

	result := new(big.Int)

	op, err := conn.ReceiveUint32()
	if err != nil {
		return nil, nil, err
	}
	if op != circuit.OpResult {
		return nil, nil, fmt.Errorf("unexpected operation: %d", op)
	}

	var label ot.Label

	for i := 0; i < prog.Outputs.Size(); i++ {
		err := conn.ReceiveLabel(&label, &labelData)
		if err != nil {
			return nil, nil, err
		}
		wire := streaming.GetInput(circuit.Wire(returnIDs[i]))
		var bit uint
		if label.Equal(wire.L0) {
			bit = 0
		} else if label.Equal(wire.L1) {
			bit = 1
		} else {
			return nil, nil, fmt.Errorf("unknown label %s for result %d",
				label, i)
		}
		result.SetBit(result, i, bit)
	}
	data := result.Bytes()
	if err := conn.SendData(data); err != nil {
		return nil, nil, err
	}
	if err := conn.Flush(); err != nil {
		return nil, nil, err
	}

	xfer = conn.Stats.Sum() - ioStats
	timing.Sample("Result", []string{circuit.FileSize(xfer).String()})

	if params.Verbose {
		timing.Print(conn.Stats)
	}

	fmt.Printf("Max permanent wires: %d, cached circuits: %d\n",
		prog.nextWireID, len(cache))
	fmt.Printf("#gates=%d (%s) #w=%d\n", prog.stats.Count(), prog.stats,
		prog.numWires)

	if params.Diagnostics {
		tab := tabulate.New(tabulate.CompactUnicodeLight)
		tab.Header("Instr").SetAlign(tabulate.ML)
		tab.Header("Count").SetAlign(tabulate.MR)
		tab.Header("XOR").SetAlign(tabulate.MR)
		tab.Header("XNOR").SetAlign(tabulate.MR)
		tab.Header("AND").SetAlign(tabulate.MR)
		tab.Header("OR").SetAlign(tabulate.MR)
		tab.Header("INV").SetAlign(tabulate.MR)
		tab.Header("!XOR").SetAlign(tabulate.MR)
		tab.Header("L").SetAlign(tabulate.MR)
		tab.Header("W").SetAlign(tabulate.MR)

		var keys []string
		for k := range istats {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return istats[keys[i]].Cost() > istats[keys[j]].Cost()
		})

		for _, key := range keys {
			stats := istats[key]
			if stats.Count() > 0 {
				row := tab.Row()
				row.Column(key)
				row.Column(fmt.Sprintf("%d", stats[circuit.Count]))
				row.Column(fmt.Sprintf("%d", stats[circuit.XOR]))
				row.Column(fmt.Sprintf("%d", stats[circuit.XNOR]))
				row.Column(fmt.Sprintf("%d", stats[circuit.AND]))
				row.Column(fmt.Sprintf("%d", stats[circuit.OR]))
				row.Column(fmt.Sprintf("%d", stats[circuit.INV]))
				row.Column(fmt.Sprintf("%d",
					stats[circuit.OR]+stats[circuit.AND]+stats[circuit.INV]))
				row.Column(fmt.Sprintf("%d", stats[circuit.NumLevels]))
				row.Column(fmt.Sprintf("%d", stats[circuit.MaxWidth]))
			}
		}
		tab.Print(os.Stdout)
	}

	return prog.Outputs, prog.Outputs.Split(result), nil
}

func addStats(istats map[string]circuit.Stats, instr Instr,
	circ *circuit.Circuit) {

	var max types.Size
	for _, in := range instr.In {
		if in.Type.Bits > max {
			max = in.Type.Bits
		}
	}
	key := fmt.Sprintf("%s/%d", instr.Op, max)
	stats, ok := istats[key]
	if !ok {
		stats = circuit.Stats{}
	}
	stats.Add(circ.Stats)
	istats[key] = stats
}

func (prog *Program) garble(conn *p2p.Conn, streaming *circuit.Streaming,
	step int, circ *circuit.Circuit, in, out []circuit.Wire) error {

	var maxID circuit.Wire
	for _, id := range in {
		if id > maxID {
			maxID = id
		}
	}
	for _, id := range out {
		if id > maxID {
			maxID = id
		}
	}

	if err := conn.SendUint32(circuit.OpCircuit); err != nil {
		return err
	}
	if err := conn.SendUint32(step); err != nil {
		return err
	}
	if err := conn.SendUint32(circ.NumGates); err != nil {
		return err
	}
	if err := conn.SendUint32(circ.NumWires); err != nil {
		return err
	}
	if err := conn.SendUint32(int(maxID + 1)); err != nil {
		return err
	}

	gStart := time.Now()
	err := streaming.Garble(circ, in, out)
	if err != nil {
		return err
	}
	dt := time.Now().Sub(gStart)
	elapsed := float64(time.Now().UnixNano() - gStart.UnixNano())
	elapsed /= 1000000000
	if elapsed > 0 && false {
		fmt.Printf("%05d: - garbled %10.0f gates/s (%s)\n",
			step, float64(circ.NumGates)/elapsed, dt)
	}
	prog.garbleDuration += dt
	prog.stats.Add(circ.Stats)
	prog.numWires += circ.NumWires

	return nil
}

// ZeroWire returns a wire with value 0.
func (prog *Program) ZeroWire(conn *p2p.Conn, streaming *circuit.Streaming) (
	*circuits.Wire, error) {

	if prog.zeroWire == nil {
		wires, err := prog.AssignedWires("{zero}", 1)
		if err != nil {
			return nil, err
		}
		err = prog.garble(conn, streaming, 0, &circuit.Circuit{
			NumGates: 1,
			NumWires: 2,
			Inputs: []circuit.IOArg{
				{
					Name: "i0",
					Type: "uint1",
					Size: 1,
				},
			},
			Outputs: []circuit.IOArg{
				{
					Name: "o0",
					Type: "uint1",
					Size: 1,
				},
			},
			Gates: []circuit.Gate{
				{
					Input0: 0,
					Input1: 0,
					Output: 1,
					Op:     circuit.XOR,
				},
			},
			Stats: circuit.Stats{
				circuit.XOR: 1,
			},
		}, []circuit.Wire{0}, []circuit.Wire{circuit.Wire(wires[0].ID())})
		if err != nil {
			return nil, err
		}
		prog.zeroWire = wires[0]
	}
	return prog.zeroWire, nil
}

// OneWire returns wire with value 1.
func (prog *Program) OneWire(conn *p2p.Conn, streaming *circuit.Streaming) (
	*circuits.Wire, error) {

	if prog.oneWire == nil {
		wires, err := prog.AssignedWires("{one}", 1)
		if err != nil {
			return nil, err
		}
		err = prog.garble(conn, streaming, 0, &circuit.Circuit{
			NumGates: 1,
			NumWires: 2,
			Inputs: []circuit.IOArg{
				{
					Name: "i0",
					Type: "uint1",
					Size: 1,
				},
			},
			Outputs: []circuit.IOArg{
				{
					Name: "o0",
					Type: "uint1",
					Size: 1,
				},
			},
			Gates: []circuit.Gate{
				{
					Input0: 0,
					Input1: 0,
					Output: 1,
					Op:     circuit.XNOR,
				},
			},
			Stats: circuit.Stats{
				circuit.XNOR: 1,
			},
		}, []circuit.Wire{0}, []circuit.Wire{circuit.Wire(wires[0].ID())})
		if err != nil {
			return nil, err
		}
		prog.oneWire = wires[0]
	}
	return prog.oneWire, nil
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

// NewCircuit creates a new circuit.
type NewCircuit func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) (cacheable bool, err error)

// NewBinary creates a new binary circuit.
type NewBinary func(cc *circuits.Compiler, a, b []*circuits.Wire,
	out []*circuits.Wire) error

func newBinary(bin NewBinary) NewCircuit {
	return func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) (bool, error) {
		return true, bin(cc, in[0], in[1], out)
	}
}

func newMultiplier(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) (bool, error) {
	return true, circuits.NewMultiplier(cc, cc.Params.CircMultArrayTreshold,
		in[0], in[1], out)
}

func newDivider(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) (bool, error) {
	return true, circuits.NewDivider(cc, in[0], in[1], out, nil)
}

func newModulo(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) (bool, error) {
	return true, circuits.NewDivider(cc, in[0], in[1], nil, out)
}

func newIndex(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) (bool, error) {
	offset, err := instr.In[1].ConstInt()
	if err != nil {
		return false, fmt.Errorf("%s: unsupported offset type %T: %s",
			instr.Op, instr.In[1], err)
	}
	return true, circuits.NewIndex(cc, int(instr.In[0].Type.ElementType.Bits),
		in[0][offset:], in[2], out)
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
	Index: newIndex,
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
		out []*circuits.Wire) (bool, error) {
		return true, instr.Builtin(cc, in[0], in[1], out)
	},
	Phi: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) (bool, error) {
		return true, circuits.NewMUX(cc, in[0], in[1], in[2], out)
	},
	Bts: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) (bool, error) {
		index, err := instr.In[1].ConstInt()
		if err != nil {
			return false,
				fmt.Errorf("%s unsupported index type %T: %s",
					instr.Op, instr.In[1], err)
		}
		return false, circuits.NewBitSetTest(cc, in[0], index, out)
	},
	Btc: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) (bool, error) {
		index, err := instr.In[1].ConstInt()
		if err != nil {
			return false,
				fmt.Errorf("%s unsupported index type %T: %s",
					instr.Op, instr.In[1], err)
		}
		return false, circuits.NewBitClrTest(cc, in[0], index, out)
	},
}
