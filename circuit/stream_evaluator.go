//
// Copyright (c) 2020-2025 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
	"github.com/markkurossi/mpc/types"
)

// Protocol operation codes.
const (
	OpResult = iota
	OpCircuit
	OpReturn
)

// StreamEval is a streaming garbled circuit evaluator.
type StreamEval struct {
	key   []byte
	alg   cipher.Block
	wires [][]ot.Label
	tmp   []ot.Label
}

// NewStreamEval creates a new streaming garbled circuit evaluator.
func NewStreamEval(key []byte, numInputs, numOutputs int) (*StreamEval, error) {
	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := &StreamEval{
		key: key,
		alg: alg,
	}

	stream.ensureWires(numInputs + numOutputs)

	return stream, nil
}

func (stream *StreamEval) wire(w Wire) ot.Label {
	return stream.wires[w>>16][w&0xffff]
}

func (stream *StreamEval) setWire(w Wire, label ot.Label) {
	stream.wires[w>>16][w&0xffff] = label
}

// Get gets the value of the wire.
func (stream *StreamEval) Get(tmp bool, w Wire) ot.Label {
	if tmp {
		return stream.tmp[w]
	}
	return stream.wire(w)
}

// SetInputs sets the specified input wire range.
func (stream *StreamEval) SetInputs(offset int, inputs []ot.Label) {
	for i := 0; i < len(inputs); i++ {
		stream.setWire(Wire(offset+i), inputs[i])
	}
}

// Set sets the value of the wire.
func (stream *StreamEval) Set(tmp bool, w Wire, label ot.Label) {
	if tmp {
		stream.tmp[w] = label
	} else {
		stream.setWire(w, label)
	}
}

// InitCircuit initializes the stream evaluator with wires.
func (stream *StreamEval) InitCircuit(numWires, numTmpWires int) {
	stream.ensureWires(numWires)
	if len(stream.tmp) < numTmpWires {
		stream.tmp = make([]ot.Label, numTmpWires)
	}
}

func (stream *StreamEval) ensureWires(max int) {
	for len(stream.wires)*0x10000 <= max {
		stream.wires = append(stream.wires, make([]ot.Label, 0x10000))
	}
}

// StreamEvaluator runs the stream evaluator on the connection. The
// evaluator's input values must be specified as a string array in
// inputFlag or as a value array in inputValues.
func StreamEvaluator(conn *p2p.Conn, oti ot.OT, inputFlag []string,
	inputValues []interface{}, verbose bool) (IO, []*big.Int, error) {

	timing := NewTiming()

	// Receive program info.
	if verbose {
		fmt.Printf(" - Waiting for program info...\n")
	}
	key, err := conn.ReceiveData()
	if err != nil {
		return nil, nil, err
	}
	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	// Peer input.
	in1, err := receiveArgument(conn)
	if err != nil {
		return nil, nil, err
	}
	// Our input.
	in2, err := receiveArgument(conn)
	if err != nil {
		return nil, nil, err
	}
	var inputs *big.Int
	if len(inputFlag) > 0 {
		inputs, err = in2.Parse(inputFlag)
	} else {
		inputs, err = in2.Set(nil, inputValues)
	}
	if err != nil {
		return nil, nil, err
	}
	// Program outputs.
	numOutputs, err := conn.ReceiveUint32()
	if err != nil {
		return nil, nil, err
	}
	var outputs IO
	for i := 0; i < numOutputs; i++ {
		out, err := receiveArgument(conn)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, out)
	}

	numSteps, err := conn.ReceiveUint32()
	if err != nil {
		return nil, nil, err
	}

	if verbose {
		fmt.Printf(" - In1: %s\n", in1)
		fmt.Printf(" + In2: %s\n", in2)
		fmt.Printf(" - Out: %s\n", outputs)
		fmt.Printf(" -  In: %s\n", inputFlag)
	}

	streaming, err := NewStreamEval(key, int(in1.Type.Bits+in2.Type.Bits),
		outputs.Size())
	if err != nil {
		return nil, nil, err
	}

	// Receive peer inputs.
	var label ot.Label
	var labelData ot.LabelData
	for w := Wire(0); w < Wire(in1.Type.Bits); w++ {
		err := conn.ReceiveLabel(&label, &labelData)
		if err != nil {
			return nil, nil, err
		}
		streaming.Set(false, w, label)
	}

	// Init oblivious transfer.
	err = oti.InitReceiver(conn)
	if err != nil {
		return nil, nil, err
	}
	ioStats := conn.Stats.Sum()
	timing.Sample("Init", []string{FileSize(ioStats).String()})

	// Query our inputs.
	if verbose {
		fmt.Printf(" - Querying our inputs...\n")
	}
	flags := make([]bool, in2.Type.Bits)
	for i := 0; i < int(in2.Type.Bits); i++ {
		if inputs.Bit(i) == 1 {
			flags[i] = true
		}
	}
	inputLabels := make([]ot.Label, int(in2.Type.Bits))
	if err := oti.Receive(flags, inputLabels); err != nil {
		return nil, nil, err
	}
	streaming.SetInputs(int(in1.Type.Bits), inputLabels)

	xfer := conn.Stats.Sum() - ioStats
	ioStats = conn.Stats.Sum()
	timing.Sample("Inputs", []string{FileSize(xfer).String()})

	ws := func(i int, tmp bool) string {
		if tmp {
			return fmt.Sprintf("~%d", i)
		}
		return fmt.Sprintf("w%d", i)
	}

	// Evaluate program.
	if verbose {
		fmt.Printf(" - Evaluating program...\n")
	}
	var garbled [4]ot.Label
	var lastStep int

	var rawResult *big.Int

	start := time.Now()
	lastReport := start
loop:
	for {
		op, err := conn.ReceiveUint32()
		if err != nil {
			return nil, nil, err
		}
		switch op {
		case OpCircuit:
			step, err := conn.ReceiveUint32()
			if err != nil {
				return nil, nil, err
			}
			numGates, err := conn.ReceiveUint32()
			if err != nil {
				return nil, nil, err
			}
			numTmpWires, err := conn.ReceiveUint32()
			if err != nil {
				return nil, nil, err
			}
			numWires, err := conn.ReceiveUint32()
			if err != nil {
				return nil, nil, err
			}
			if step-lastStep >= 10 && verbose {
				lastStep = step
				now := time.Now()
				if now.Sub(lastReport) > time.Second*5 {
					lastReport = now
					elapsed := time.Since(start)
					done := float64(step) / float64(numSteps)
					if done > 0 {
						total := time.Duration(float64(elapsed) / done)
						progress := fmt.Sprintf("%d/%d", step, numSteps)
						remaining := fmt.Sprintf("%24s", total-elapsed)
						fmt.Printf("%-14s\t%s remaining\tETA %s\n",
							progress, remaining,
							start.Add(total).Format(time.Stamp))
					} else {
						fmt.Printf("%d/%d\n", step, numSteps)
					}
				}
			}
			streaming.InitCircuit(numWires, numTmpWires)
			var id uint32
			for i := 0; i < numGates; i++ {
				gop, err := conn.ReceiveByte()
				if err != nil {
					return nil, nil, err
				}
				var aTmp, bTmp, cTmp bool
				if gop&0b10000000 != 0 {
					aTmp = true
				}
				if gop&0b01000000 != 0 {
					bTmp = true
				}
				if gop&0b00100000 != 0 {
					cTmp = true
				}
				var recvWire func() (int, error)
				if gop&0b00010000 != 0 {
					recvWire = conn.ReceiveUint16
				} else {
					recvWire = conn.ReceiveUint32
				}

				gop &^= 0b11110000

				var aIndex, bIndex, cIndex int
				var tableCount int

				switch Operation(gop) {
				case XOR, XNOR, AND, OR:
					aIndex, err = recvWire()
					if err != nil {
						return nil, nil, err
					}
					bIndex, err = recvWire()
					if err != nil {
						return nil, nil, err
					}
					cIndex, err = recvWire()
					if err != nil {
						return nil, nil, err
					}

				case INV:
					aIndex, err = recvWire()
					if err != nil {
						return nil, nil, err
					}
					cIndex, err = recvWire()
					if err != nil {
						return nil, nil, err
					}
				default:
					return nil, nil, fmt.Errorf("invalid operation %s",
						Operation(gop))
				}
				switch Operation(gop) {
				case XOR, XNOR:
					tableCount = 0
				case INV:
					tableCount = 1
				case AND:
					tableCount = 2
				case OR:
					tableCount = 3
				}

				for c := 0; c < tableCount; c++ {
					err = conn.ReceiveLabel(&label, &labelData)
					if err != nil {
						return nil, nil, err
					}
					garbled[c] = label
				}

				var a, b, c ot.Label

				switch Operation(gop) {
				case XOR, XNOR, AND, OR:
					if StreamDebug {
						fmt.Printf("Gate%d:\t %s %s %s %s\n", i,
							ws(aIndex, aTmp), ws(bIndex, bTmp),
							Operation(gop), ws(cIndex, cTmp))
					}
					a = streaming.Get(aTmp, Wire(aIndex))
					b = streaming.Get(bTmp, Wire(bIndex))

				case INV:
					if StreamDebug {
						fmt.Printf("Gate%d:\t %s %s %s\n", i,
							ws(aIndex, aTmp), Operation(gop), ws(bIndex, bTmp))
					}
					a = streaming.Get(aTmp, Wire(aIndex))
				}

				var output ot.Label

				switch Operation(gop) {
				case XOR, XNOR:
					a.Xor(b)
					output = a

				case AND:
					if tableCount != 2 {
						return nil, nil,
							fmt.Errorf("corrupted ciruit: AND table size: %d",
								tableCount)
					}
					sa := a.S()
					sb := b.S()

					j0 := id
					j1 := id + 1
					id += 2

					tg := garbled[0]
					te := garbled[1]

					wg := encryptHalf(alg, a, j0, &labelData)
					if sa {
						wg.Xor(tg)
					}
					we := encryptHalf(alg, b, j1, &labelData)
					if sb {
						we.Xor(te)
						we.Xor(a)
					}
					output = wg
					output.Xor(we)

				case OR:
					index := idx(a, b)
					if index > 0 {
						// First row is zero and not transmitted.
						index--
						if index >= tableCount {
							return nil, nil,
								fmt.Errorf("corrupted circuit: index %d >= %d",
									index, tableCount)
						}
						c = garbled[index]
					}
					output = decrypt(alg, a, b, id, c, &labelData)
					id++

				case INV:
					index := idxUnary(a)
					if index > 0 {
						// First row is zero and not transmitted.
						index--
						if index >= tableCount {
							return nil, nil,
								fmt.Errorf("corrupted circuit: index %d >= %d",
									index, tableCount)
						}
						c = garbled[index]
					}

					output = decrypt(alg, a, b, id, c, &labelData)
					id++
				}
				streaming.Set(cTmp, Wire(cIndex), output)
			}

		case OpReturn:
			xfer := conn.Stats.Sum() - ioStats
			ioStats = conn.Stats.Sum()
			timing.Sample("Eval", []string{FileSize(xfer).String()})

			var labels []ot.Label
			for i := 0; i < outputs.Size(); i++ {
				id, err := conn.ReceiveUint32()
				if err != nil {
					return nil, nil, err
				}
				label := streaming.Get(false, Wire(id))
				labels = append(labels, label)
			}

			// Resolve result values.
			if err := conn.SendUint32(OpResult); err != nil {
				return nil, nil, err
			}
			var labelData ot.LabelData
			for _, l := range labels {
				if err := conn.SendLabel(l, &labelData); err != nil {
					return nil, nil, err
				}
			}
			if err := conn.Flush(); err != nil {
				return nil, nil, err
			}

			result, err := conn.ReceiveData()
			if err != nil {
				return nil, nil, err
			}
			rawResult = new(big.Int).SetBytes(result)
			break loop

		default:
			return nil, nil, fmt.Errorf("unknown operation %d", op)
		}
	}

	xfer = conn.Stats.Sum() - ioStats
	timing.Sample("Result", []string{FileSize(xfer).String()})

	if verbose {
		timing.Print(conn.Stats)
	}

	return outputs, outputs.Split(rawResult), nil
}

func receiveArgument(conn *p2p.Conn) (arg IOArg, err error) {
	name, err := conn.ReceiveString()
	if err != nil {
		return arg, err
	}
	t, err := conn.ReceiveString()
	if err != nil {
		return arg, err
	}
	size, err := conn.ReceiveUint32()
	if err != nil {
		return arg, err
	}
	arg.Name = name
	arg.Type, err = types.Parse(t)
	if err != nil {
		return arg, err
	}
	arg.Type.Bits = types.Size(size)

	if arg.Type.Type == types.TSlice {
		arg.Type.ArraySize = arg.Type.Bits / arg.Type.ElementType.Bits
	}

	count, err := conn.ReceiveUint32()
	if err != nil {
		return arg, err
	}
	for i := 0; i < count; i++ {
		a, err := receiveArgument(conn)
		if err != nil {
			return arg, err
		}
		arg.Compound = append(arg.Compound, a)
	}
	return arg, nil
}
