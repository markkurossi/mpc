//
// Copyright (c) 2022 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"crypto/rand"

	"github.com/markkurossi/mpc/ot"
)

// F implements Fgc computation.
type F struct {
	p1OT *ot.Sender
	p2OT *ot.Receiver
}

// NewF creates a new Fgc with oblivious-transfer with keyBits bits.
func NewF(keyBits int) (*F, error) {
	p1OT, err := ot.NewSender(keyBits)
	if err != nil {
		return nil, err
	}
	p2OT, err := ot.NewReceiver(p1OT.PublicKey())
	if err != nil {
		return nil, err
	}
	return &F{
		p1OT: p1OT,
		p2OT: p2OT,
	}, nil
}

// X implements secure multiplication so that c⊕d = a⋅b.
func (f *F) X(a, b uint) (c, d uint, err error) {
	// P1 chooses a random r e {0, 1}
	var buf [1]byte
	_, err = rand.Read(buf[:])
	if err != nil {
		return
	}

	r := uint(buf[0] & 0x1)

	x0 := r
	x1 := r ^ a

	var p1Xfer *ot.SenderXfer
	p1Xfer, err = f.p1OT.NewTransfer([]byte{byte(x0)}, []byte{byte(x1)})
	if err != nil {
		return
	}

	var p2Xfer *ot.ReceiverXfer
	p2Xfer, err = f.p2OT.NewTransfer(b)
	if err != nil {
		return
	}

	// OT
	err = p2Xfer.ReceiveRandomMessages(p1Xfer.RandomMessages())
	if err != nil {
		return
	}
	p1Xfer.ReceiveV(p2Xfer.V())
	err = p2Xfer.ReceiveMessages(p1Xfer.Messages())
	if err != nil {
		return
	}

	xb, _ := p2Xfer.Message()

	return r, uint(xb[0]), nil
}
