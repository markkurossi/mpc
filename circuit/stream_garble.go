//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

const (
	// StreamDebug controls the debugging output of the streaming
	// garbling.
	StreamDebug = false
)

// Streaming is a streaming garbled circuit garbler.
type Streaming struct {
	conn     *p2p.Conn
	key      []byte
	alg      cipher.Block
	r        ot.Label
	wires    []ot.Wire
	tmp      []ot.Wire
	in       []Wire
	out      []Wire
	firstTmp Wire
	firstOut Wire
}

// NewStreaming creates a new streaming garbled circuit garbler.
func NewStreaming(key []byte, inputs []Wire, conn *p2p.Conn) (
	*Streaming, error) {

	r, err := ot.NewLabel(rand.Reader)
	if err != nil {
		return nil, err
	}
	r.SetS(true)

	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := &Streaming{
		conn: conn,
		key:  key,
		alg:  alg,
		r:    r,
	}

	stream.ensureWires(inputs)

	// Assing all input wires.
	for i := 0; i < len(inputs); i++ {
		w, err := makeLabels(stream.r)
		if err != nil {
			return nil, err
		}
		stream.wires[inputs[i]] = w
	}

	return stream, nil
}

func (stream *Streaming) ensureWires(wires []Wire) {
	// Verify that wires is big enough.
	var max Wire
	for _, w := range wires {
		if w > max {
			max = w
		}
	}
	if len(stream.wires) <= int(max) {
		var i int
		for i = 1024; i <= int(max); i <<= 1 {
		}
		n := make([]ot.Wire, i)
		copy(n, stream.wires)
		stream.wires = n
	}
}

func (stream *Streaming) initCircuit(c *Circuit, in, out []Wire) {
	stream.ensureWires(in)
	stream.ensureWires(out)

	if len(stream.tmp) < c.NumWires {
		stream.tmp = make([]ot.Wire, c.NumWires)
	}

	stream.in = in
	stream.out = out

	stream.firstTmp = Wire(len(in))
	stream.firstOut = Wire(c.NumWires - len(out))
}

// GetInput gets the value of the input wire.
func (stream *Streaming) GetInput(w Wire) ot.Wire {
	return stream.wires[w]
}

// Get gets the value of the wire.
func (stream *Streaming) Get(w Wire) (ot.Wire, Wire, bool) {
	if w < stream.firstTmp {
		index := stream.in[w]
		return stream.wires[index], index, false
	} else if w >= stream.firstOut {
		index := stream.out[w-stream.firstOut]
		return stream.wires[index], index, false
	} else {
		return stream.tmp[w], w, true
	}
}

// Set sets the value of the wire.
func (stream *Streaming) Set(w Wire, val ot.Wire) (index Wire, tmp bool) {
	if w < stream.firstTmp {
		index = stream.in[w]
		stream.wires[index] = val
	} else if w >= stream.firstOut {
		index = stream.out[w-stream.firstOut]
		stream.wires[index] = val
	} else {
		index = w
		tmp = true
		stream.tmp[w] = val
	}
	return index, tmp
}

// Garble garbles the circuit and streams the garbled tables into the
// stream.
func (stream *Streaming) Garble(c *Circuit, in, out []Wire) error {
	if StreamDebug {
		fmt.Printf(" - Streaming.Garble: in=%v, out=%v\n", in, out)
	}

	stream.initCircuit(c, in, out)

	// Garble gates.
	var data ot.LabelData
	buf := make([]ot.Label, 4)
	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]
		err := stream.GarbleGate(gate, uint32(i), buf, &data)
		if err != nil {
			return err
		}
	}
	return nil
}

// GarbleGate garbles the gate and streams it to the stream.
func (stream *Streaming) GarbleGate(g *Gate, id uint32,
	table []ot.Label, data *ot.LabelData) error {

	var a, b, c ot.Wire
	var aIndex, bIndex, cIndex Wire
	var aTmp, bTmp, cTmp bool
	var err error

	table = table[0:4]
	var tableCount, wireCount int

	// Inputs.
	switch g.Op {
	case XOR, XNOR, AND, OR:
		b, bIndex, bTmp = stream.Get(g.Input1)
		fallthrough

	case INV:
		a, aIndex, aTmp = stream.Get(g.Input0)

	default:
		return fmt.Errorf("invalid gate type %s", g.Op)
	}

	// Output.
	switch g.Op {
	case XOR:
		l0 := a.L0
		l0.Xor(b.L0)

		l1 := l0
		l1.Xor(stream.r)
		c = ot.Wire{
			L0: l0,
			L1: l1,
		}

	case XNOR:
		l0 := a.L0
		l0.Xor(b.L0)

		l1 := l0
		l1.Xor(stream.r)
		c = ot.Wire{
			L0: l1,
			L1: l0,
		}

	case AND:
		pa := a.L0.S()
		pb := b.L0.S()

		// XXX need two indices here, must communicate back how many
		// we used.
		j0 := id
		j1 := id + 1

		// First half gate.
		tg := encryptHalf(stream.alg, a.L0, j0, data)
		tg.Xor(encryptHalf(stream.alg, a.L1, j0, data))
		if pb {
			tg.Xor(stream.r)
		}
		wg0 := encryptHalf(stream.alg, a.L0, j0, data)
		if pa {
			wg0.Xor(tg)
		}

		// Second half gate.
		te := encryptHalf(stream.alg, b.L0, j1, data)
		te.Xor(encryptHalf(stream.alg, b.L1, j1, data))
		te.Xor(a.L0)
		we0 := encryptHalf(stream.alg, b.L0, j1, data)
		if pb {
			we0.Xor(te)
			we0.Xor(a.L0)
		}

		// Combine halves
		l0 := wg0
		l0.Xor(we0)

		l1 := l0
		l1.Xor(stream.r)

		c = ot.Wire{
			L0: l0,
			L1: l1,
		}
		table[0] = tg
		table[1] = te
		tableCount = 2

	default:
		c, err = makeLabels(stream.r)
		if err != nil {
			return err
		}
	}

	ws := func(i Wire, tmp bool) string {
		if tmp {
			return fmt.Sprintf("~%d", i)
		}
		return i.String()
	}

	cIndex, cTmp = stream.Set(g.Output, c)
	if StreamDebug && false {
		fmt.Printf("Set %s\n", ws(cIndex, cTmp))
	}

	switch g.Op {
	case XOR, XNOR:
		// Free XOR.
		wireCount = 3

	case AND:
		// Half AND garbled above.
		wireCount = 3

	case OR:
		// a b c
		// -----
		// 0 0 0
		// 0 1 1
		// 1 0 1
		// 1 1 1
		table[idx(a.L0, b.L0)] = encrypt(stream.alg, a.L0, b.L0, c.L0, id, data)
		table[idx(a.L0, b.L1)] = encrypt(stream.alg, a.L0, b.L1, c.L1, id, data)
		table[idx(a.L1, b.L0)] = encrypt(stream.alg, a.L1, b.L0, c.L1, id, data)
		table[idx(a.L1, b.L1)] = encrypt(stream.alg, a.L1, b.L1, c.L1, id, data)
		tableCount = 4
		wireCount = 3

	case INV:
		// a b c
		// -----
		// 0   1
		// 1   0
		zero := ot.Label{}
		table[idxUnary(a.L0)] = encrypt(stream.alg, a.L0, zero, c.L1, id, data)
		table[idxUnary(a.L1)] = encrypt(stream.alg, a.L1, zero, c.L0, id, data)
		tableCount = 2
		wireCount = 2

	default:
		return fmt.Errorf("invalid operand %s", g.Op)
	}

	op := byte(g.Op)
	if aTmp {
		op |= 0b10000000
	}
	if bTmp {
		op |= 0b01000000
	}
	if cTmp {
		op |= 0b00100000
	}
	var sendWire func(w int) error
	if aIndex <= 0xffff && bIndex <= 0xffff && cIndex <= 0xffff {
		op |= 0b00010000
		sendWire = stream.conn.SendUint16
	} else {
		sendWire = stream.conn.SendUint32
	}

	if err := stream.conn.SendByte(op); err != nil {
		return err
	}
	switch wireCount {
	case 3:
		if err := sendWire(int(aIndex)); err != nil {
			return err
		}
		if err := sendWire(int(bIndex)); err != nil {
			return err
		}
		if err := sendWire(int(cIndex)); err != nil {
			return err
		}
		if StreamDebug {
			fmt.Printf(" - Gate%d:\t%s %s %s %s\n", id,
				ws(aIndex, aTmp), ws(bIndex, bTmp),
				g.Op, ws(cIndex, cTmp))
		}

	case 2:
		if err := sendWire(int(aIndex)); err != nil {
			return err
		}
		if err := sendWire(int(cIndex)); err != nil {
			return err
		}
		if StreamDebug {
			fmt.Printf("Gate%d:\t%s %s %s\n", id,
				ws(aIndex, aTmp), g.Op, ws(cIndex, cTmp))
		}

	default:
		panic(fmt.Sprintf("invalid wire count: %d", wireCount))
	}

	for i := 0; i < tableCount; i++ {
		if err := stream.conn.SendLabel(table[i], data); err != nil {
			return err
		}
	}

	return nil
}
