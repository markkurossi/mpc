//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bufio"
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
)

type FileSize uint64

func (s FileSize) String() string {
	if s > 1000*1000*1000*1000 {
		return fmt.Sprintf("%d TB", s/(1000*1000*1000*1000))
	} else if s > 1000*1000*1000 {
		return fmt.Sprintf("%d GB", s/(1000*1000*1000))
	} else if s > 1000*1000 {
		return fmt.Sprintf("%d MB", s/(1000*1000))
	} else if s > 1000 {
		return fmt.Sprintf("%d kB", s/1000)
	} else {
		return fmt.Sprintf("%d B", s)
	}
}

func Garbler(conn *bufio.ReadWriter, circ *Circuit, input *big.Int,
	key []byte, verbose bool) (*big.Int, error) {

	start := time.Now()

	garbled, err := circ.Garble(key)
	if err != nil {
		return nil, err
	}

	t := time.Now()
	if verbose {
		fmt.Printf("Garble:\t%s\n", t.Sub(start))
	}
	start = t

	// Send garbled tables.
	var size FileSize
	for id, data := range garbled.Gates {
		if err := sendUint32(conn, id); err != nil {
			return nil, err
		}
		size += 4
		if err := sendUint32(conn, len(data)); err != nil {
			return nil, err
		}
		size += 4
		for _, d := range data {
			if err := sendData(conn, d); err != nil {
				return nil, err
			}
			size += FileSize(4 + len(d))
		}
	}

	// Select our inputs.
	var n1 [][]byte
	for i := 0; i < circ.N1; i++ {
		wire := garbled.Wires[i]

		var n []byte

		if input.Bit(i) == 1 {
			n = wire.Label1.Bytes()
		} else {
			n = wire.Label0.Bytes()
		}
		n1 = append(n1, n)
	}

	// Send our inputs.
	for idx, i := range n1 {
		if verbose {
			fmt.Printf("N1[%d]:\t%x\n", idx, i)
		}
		if err := sendData(conn, i); err != nil {
			return nil, err
		}
		size += FileSize(4 + len(i))
	}

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048, garbled.Wires)
	if err != nil {
		return nil, err
	}

	// Send our public key.
	pub := sender.PublicKey()
	data := pub.N.Bytes()
	if err := sendData(conn, data); err != nil {
		return nil, err
	}
	size += FileSize(4 + len(data))
	if err := sendUint32(conn, pub.E); err != nil {
		return nil, err
	}
	size += 4
	conn.Flush()
	t = time.Now()
	if verbose {
		fmt.Printf("Xfer:\t%s\t%s\n", t.Sub(start), size)
	}
	start = t

	// Process messages.

	var xfer *ot.SenderXfer
	lastOT := start
	done := false
	result := big.NewInt(0)

	for !done {
		op, err := receiveUint32(conn)
		if err != nil {
			return nil, err
		}
		switch op {
		case OP_OT:
			bit, err := receiveUint32(conn)
			if err != nil {
				return nil, err
			}
			xfer, err = sender.NewTransfer(bit)
			if err != nil {
				return nil, err
			}

			x0, x1 := xfer.RandomMessages()
			if err := sendData(conn, x0); err != nil {
				return nil, err
			}
			if err := sendData(conn, x1); err != nil {
				return nil, err
			}
			conn.Flush()

			v, err := receiveData(conn)
			if err != nil {
				return nil, err
			}
			xfer.ReceiveV(v)

			m0p, m1p, err := xfer.Messages()
			if err != nil {
				return nil, err
			}
			if err := sendData(conn, m0p); err != nil {
				return nil, err
			}
			if err := sendData(conn, m1p); err != nil {
				return nil, err
			}
			conn.Flush()
			lastOT = time.Now()

		case OP_RESULT:

			for i := 0; i < circ.N3; i++ {
				label, err := receiveData(conn)
				if err != nil {
					return nil, err
				}
				wire := garbled.Wires[circ.NumWires-circ.N3+i]

				var bit uint
				if bytes.Compare(label, wire.Label0.Bytes()) == 0 {
					bit = 0
				} else if bytes.Compare(label, wire.Label1.Bytes()) == 0 {
					bit = 1
				} else {
					return nil, fmt.Errorf("Unknown label %x for result %d",
						label, i)
				}
				result = big.NewInt(0).SetBit(result, i, bit)
			}
			if err := sendData(conn, result.Bytes()); err != nil {
				return nil, err
			}
			conn.Flush()
			done = true
		}
	}
	t = time.Now()
	if verbose {
		fmt.Printf("OT:\t%s\n", lastOT.Sub(start))
		fmt.Printf("Eval:\t%s\n", t.Sub(lastOT))
	}
	start = t

	return result, nil
}
