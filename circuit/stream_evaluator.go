//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

type StreamEval struct {
	key   []byte
	alg   cipher.Block
	wires []ot.Label
	tmp   []ot.Label
}

func NewStreamEval(key []byte, numInputs, numOutputs int) (*StreamEval, error) {
	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &StreamEval{
		key:   key,
		alg:   alg,
		wires: make([]ot.Label, numInputs+numOutputs),
	}, nil
}

func (stream *StreamEval) Get(tmp bool, w int) ot.Label {
	if tmp {
		return stream.tmp[w]
	} else {
		return stream.wires[w]
	}
}

func (stream *StreamEval) Set(tmp bool, w int, label ot.Label) {
	if tmp {
		stream.tmp[w] = label
	} else {
		stream.wires[w] = label
	}
}

func (stream *StreamEval) InitCircuit(numWires, numTmpWires int) {
	if numWires > len(stream.wires) {
		n := make([]ot.Label, numWires)
		copy(n, stream.wires)
		stream.wires = n
	}
	if numTmpWires > len(stream.tmp) {
		stream.tmp = make([]ot.Label, numTmpWires)
	}
}

func StreamEvaluator(conn *p2p.Conn, inputFlag []string, verbose bool) (
	[]*big.Int, error) {

	timing := NewTiming()

	// Receive program info.
	if verbose {
		fmt.Printf(" - Waiting for program info...\n")
	}
	key, err := conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// Peer input.
	in1, err := receiveArgument(conn)
	if err != nil {
		return nil, err
	}
	// Our input.
	in2, err := receiveArgument(conn)
	if err != nil {
		return nil, err
	}
	inputs, err := in2.Parse(inputFlag)
	if err != nil {
		return nil, err
	}
	// Program outputs.
	numOutputs, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	var outputs IO
	for i := 0; i < numOutputs; i++ {
		out, err := receiveArgument(conn)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, out)
	}

	numSteps, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}

	fmt.Printf(" - In1: %s\n", in1)
	fmt.Printf(" + In2: %s\n", in2)
	fmt.Printf(" - Out: %s\n", outputs)
	fmt.Printf(" -  In: %s\n", inputFlag)

	streaming, err := NewStreamEval(key, in1.Size+in2.Size, outputs.Size())
	if err != nil {
		return nil, err
	}

	// Receive peer inputs.
	for w := 0; w < in1.Size; w++ {
		label, err := conn.ReceiveLabel()
		if err != nil {
			return nil, err
		}
		streaming.Set(false, w, label)
	}

	// Init oblivious transfer.
	pubN, err := conn.ReceiveData()
	if err != nil {
		return nil, err
	}
	pubE, err := conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	pub := &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(pubN),
		E: pubE,
	}
	receiver, err := ot.NewReceiver(pub)
	if err != nil {
		return nil, err
	}

	ioStats := conn.Stats
	timing.Sample("Init", []string{FileSize(ioStats.Sum()).String()})

	// Query our inputs.
	if verbose {
		fmt.Printf(" - Querying our inputs...\n")
	}
	for w := 0; w < in2.Size; w++ {
		n, err := conn.Receive(receiver, uint(in1.Size+w), inputs.Bit(w))
		if err != nil {
			return nil, err
		}
		var label ot.Label
		label.SetBytes(n)
		streaming.Set(false, in1.Size+w, label)
	}
	xfer := conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Inputs", []string{FileSize(xfer.Sum()).String()})

	ws := func(i int, tmp bool) string {
		if tmp {
			return fmt.Sprintf("~%d", i)
		} else {
			return fmt.Sprintf("w%d", i)
		}
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
			return nil, err
		}
		switch op {
		case OP_CIRCUIT:
			step, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			numGates, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			numTmpWires, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			numWires, err := conn.ReceiveUint32()
			if err != nil {
				return nil, err
			}
			if step-lastStep >= 10 && verbose {
				lastStep = step
				now := time.Now()
				if now.Sub(lastReport) > time.Second*5 {
					lastReport = now
					elapsed := time.Now().Sub(start)
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
			for i := 0; i < numGates; i++ {
				gop, err := conn.ReceiveByte()
				if err != nil {
					return nil, err
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
				var count int

				switch Operation(gop) {
				case AND, OR:
					count = 4
					fallthrough

				case XOR, XNOR:
					aIndex, err = recvWire()
					if err != nil {
						return nil, err
					}
					bIndex, err = recvWire()
					if err != nil {
						return nil, err
					}
					cIndex, err = recvWire()
					if err != nil {
						return nil, err
					}

				case INV:
					count = 2
					aIndex, err = recvWire()
					if err != nil {
						return nil, err
					}
					cIndex, err = recvWire()
					if err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("invalid operation %s",
						Operation(gop))
				}

				for c := 0; c < count; c++ {
					garbled[c], err = conn.ReceiveLabel()
					if err != nil {
						return nil, err
					}
				}

				var a, b ot.Label

				switch Operation(gop) {
				case XOR, XNOR, AND, OR:
					if StreamDebug {
						fmt.Printf("Gate%d:\t %s %s %s %s\n", i,
							ws(aIndex, aTmp), ws(bIndex, bTmp),
							Operation(gop), ws(cIndex, cTmp))
					}
					a = streaming.Get(aTmp, aIndex)
					b = streaming.Get(bTmp, bIndex)

				case INV:
					if StreamDebug {
						fmt.Printf("Gate%d:\t %s %s %s\n", i,
							ws(aIndex, aTmp), Operation(gop), ws(bIndex, bTmp))
					}
					a = streaming.Get(aTmp, aIndex)
				}

				var output ot.Label

				switch Operation(gop) {
				case XOR, XNOR:
					a.Xor(b)
					output = a

				case AND, OR:
					index := idx(a, b)
					if index >= count {
						return nil,
							fmt.Errorf("corrupted circuit: index %d >= %d",
								index, count)
					}
					output, err = decrypt(alg, a, b, uint32(i), garbled[index])
					if err != nil {
						return nil, err
					}

				case INV:
					index := idxUnary(a)
					if index >= count {
						return nil,
							fmt.Errorf("corrupted circuit: index %d >= %d",
								index, count)
					}
					output, err = decrypt(alg, a, b, uint32(i), garbled[index])
					if err != nil {
						return nil, err
					}
				}
				streaming.Set(cTmp, cIndex, output)
			}

		case OP_RETURN:
			xfer := conn.Stats.Sub(ioStats)
			ioStats = conn.Stats
			timing.Sample("Eval", []string{FileSize(xfer.Sum()).String()})

			var labels []ot.Label
			for i := 0; i < outputs.Size(); i++ {
				id, err := conn.ReceiveUint32()
				if err != nil {
					return nil, err
				}
				label := streaming.Get(false, id)
				labels = append(labels, label)
			}

			// Resolve result values.
			if err := conn.SendUint32(OP_RESULT); err != nil {
				return nil, err
			}
			for _, l := range labels {
				if err := conn.SendLabel(l); err != nil {
					return nil, err
				}
			}
			conn.Flush()

			result, err := conn.ReceiveData()
			if err != nil {
				return nil, err
			}
			rawResult = new(big.Int).SetBytes(result)
			break loop

		default:
			return nil, fmt.Errorf("unknown operation %d", op)
		}
	}

	xfer = conn.Stats.Sub(ioStats)
	ioStats = conn.Stats
	timing.Sample("Result", []string{FileSize(xfer.Sum()).String()})

	if verbose {
		timing.Print(FileSize(conn.Stats.Sum()).String())
	}

	return outputs.Split(rawResult), nil
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
	arg.Type = t
	arg.Size = size

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
