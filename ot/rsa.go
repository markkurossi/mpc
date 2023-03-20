//
// rsa.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"

	"github.com/markkurossi/crypto/pkcs1"
	"github.com/markkurossi/mpc/ot/mpint"
)

// RandomData creates size bytes of random data.
func RandomData(size int) ([]byte, error) {
	m := make([]byte, size)
	_, err := rand.Read(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Sender implements OT sender.
type Sender struct {
	key *rsa.PrivateKey
}

// NewSender creates a new OT sender for the bit.
func NewSender(keyBits int) (*Sender, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	return &Sender{
		key: key,
	}, nil
}

// MessageSize returns the maximum OT message size.
func (s *Sender) MessageSize() int {
	return s.key.PublicKey.Size()
}

// PublicKey returns the sender's public key.
func (s *Sender) PublicKey() *rsa.PublicKey {
	return &s.key.PublicKey
}

// NewTransfer creates a new OT sender data transfer.
func (s *Sender) NewTransfer(m0, m1 []byte) (*SenderXfer, error) {
	x0, err := RandomData(s.MessageSize())
	if err != nil {
		return nil, err
	}
	x1, err := RandomData(s.MessageSize())
	if err != nil {
		return nil, err
	}

	return &SenderXfer{
		sender: s,
		m0:     m0,
		m1:     m1,
		x0:     x0,
		x1:     x1,
	}, nil
}

// SenderXfer implements the OT sender data transfer.
type SenderXfer struct {
	sender *Sender
	m0     []byte
	m1     []byte
	x0     []byte
	x1     []byte
	k0     *big.Int
	k1     *big.Int
}

// MessageSize returns the maximum OT message size.
func (s *SenderXfer) MessageSize() int {
	return s.sender.MessageSize()
}

// RandomMessages creates random messages.
func (s *SenderXfer) RandomMessages() ([]byte, []byte) {
	return s.x0, s.x1
}

// ReceiveV receives the V.
func (s *SenderXfer) ReceiveV(data []byte) {
	v := mpint.FromBytes(data)
	x0 := mpint.FromBytes(s.x0)
	x1 := mpint.FromBytes(s.x1)

	s.k0 = mpint.Exp(mpint.Sub(v, x0), s.sender.key.D, s.sender.key.PublicKey.N)
	s.k1 = mpint.Exp(mpint.Sub(v, x1), s.sender.key.D, s.sender.key.PublicKey.N)
}

// Messages creates the transfer messages.
func (s *SenderXfer) Messages() ([]byte, []byte, error) {
	m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(), s.m0)
	if err != nil {
		return nil, nil, err
	}
	m0p := mpint.Add(mpint.FromBytes(m0), s.k0)

	m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(), s.m1)
	if err != nil {
		return nil, nil, err
	}
	m1p := mpint.Add(mpint.FromBytes(m1), s.k1)

	return m0p.Bytes(), m1p.Bytes(), nil
}

// Receiver implements OT receivers.
type Receiver struct {
	pub *rsa.PublicKey
}

// NewReceiver creates a new OT receiver.
func NewReceiver(pub *rsa.PublicKey) (*Receiver, error) {
	return &Receiver{
		pub: pub,
	}, nil
}

// MessageSize returns the maximum OT message size.
func (r *Receiver) MessageSize() int {
	return r.pub.Size()
}

// NewTransfer creates a new OT receiver data transfer for the bit.
func (r *Receiver) NewTransfer(bit uint) (*ReceiverXfer, error) {
	return &ReceiverXfer{
		receiver: r,
		bit:      bit,
	}, nil
}

// ReceiverXfer implements the OT receiver data transfer.
type ReceiverXfer struct {
	receiver *Receiver
	bit      uint
	k        *big.Int
	v        *big.Int
	mb       []byte
}

// ReceiveRandomMessages receives the random messages x0 and x1.
func (r *ReceiverXfer) ReceiveRandomMessages(x0, x1 []byte) error {
	k, err := rand.Int(rand.Reader, r.receiver.pub.N)
	if err != nil {
		return err
	}
	r.k = k

	var xb *big.Int
	if r.bit == 0 {
		xb = mpint.FromBytes(x0)
	} else {
		xb = mpint.FromBytes(x1)
	}

	e := big.NewInt(int64(r.receiver.pub.E))
	r.v = mpint.Mod(
		mpint.Add(xb, mpint.Exp(r.k, e, r.receiver.pub.N)), r.receiver.pub.N)

	return nil
}

// V returns the V of the exchange.
func (r *ReceiverXfer) V() []byte {
	return r.v.Bytes()
}

// ReceiveMessages processes the received m0p and m1p messages.
func (r *ReceiverXfer) ReceiveMessages(m0p, m1p []byte, err error) error {
	if err != nil {
		return err
	}
	var mbp *big.Int
	if r.bit == 0 {
		mbp = mpint.FromBytes(m0p)
	} else {
		mbp = mpint.FromBytes(m1p)
	}
	mbBytes := make([]byte, r.receiver.MessageSize())
	mbIntBytes := mpint.Sub(mbp, r.k).Bytes()
	ofs := len(mbBytes) - len(mbIntBytes)
	copy(mbBytes[ofs:], mbIntBytes)

	mb, err := pkcs1.ParseEncryptionBlock(mbBytes)
	if err != nil {
		return err
	}
	r.mb = mb

	return nil
}

// Message returns the message and bit from the exchange.
func (r *ReceiverXfer) Message() (m []byte, bit uint) {
	return r.mb, r.bit
}

// RSA implements RSA OT as the OT interface.
type RSA struct {
	keyBits int
	name    string
	io      IO
	priv    *rsa.PrivateKey
	pub     *rsa.PublicKey
}

// NewRSA creates a new RSA OT implementing the OT interface. The
// argument specifies the RSA key size in bits.
func NewRSA(keyBits int) *RSA {
	return &RSA{
		keyBits: keyBits,
		name:    fmt.Sprintf("RSA-%v", keyBits),
	}
}

func (r *RSA) messageSize() int {
	return (r.keyBits + 7) / 8
}

// InitSender initializes the OT sender.
func (r *RSA) InitSender(io IO) error {
	r.io = io

	priv, err := rsa.GenerateKey(rand.Reader, r.keyBits)
	if err != nil {
		return err
	}
	r.priv = priv
	r.pub = &priv.PublicKey

	if err := SendString(io, r.name); err != nil {
		return err
	}
	if err := io.SendData(r.pub.N.Bytes()); err != nil {
		return err
	}
	if err := io.SendUint32(r.pub.E); err != nil {
		return err
	}

	return io.Flush()
}

// InitReceiver initializes the OT receiver.
func (r *RSA) InitReceiver(io IO) error {
	r.io = io

	name, err := ReceiveString(io)
	if err != nil {
		return err
	}
	if name != r.name {
		return fmt.Errorf("invalid algorithm %s, expected %s", name, r.name)
	}
	pubN, err := ReceiveBigInt(io)
	if err != nil {
		return err
	}
	pubE, err := io.ReceiveUint32()
	if err != nil {
		return err
	}
	r.pub = &rsa.PublicKey{
		N: pubN,
		E: pubE,
	}
	return nil
}

// Send sends the wire labels with OT.
func (r *RSA) Send(wires []Wire) error {
	for i := 0; i < len(wires); i++ {
		// Send random messages.
		x0, err := RandomData(r.messageSize())
		if err != nil {
			return err
		}
		x1, err := RandomData(r.messageSize())
		if err != nil {
			return err
		}
		if err := r.io.SendData(x0); err != nil {
			return err
		}
		if err := r.io.SendData(x1); err != nil {
			return err
		}
		if err := r.io.Flush(); err != nil {
			return err
		}

		// Receive V.
		v, err := ReceiveBigInt(r.io)
		if err != nil {
			return err
		}
		x0i := mpint.FromBytes(x0)
		x1i := mpint.FromBytes(x1)
		k0 := mpint.Exp(mpint.Sub(v, x0i), r.priv.D, r.pub.N)
		k1 := mpint.Exp(mpint.Sub(v, x1i), r.priv.D, r.pub.N)

		// Create transfer messages.
		var ld LabelData
		wires[i].L0.GetData(&ld)
		m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, r.messageSize(), ld[:])
		if err != nil {
			return err
		}
		m0p := mpint.Add(mpint.FromBytes(m0), k0)
		if err := r.io.SendData(m0p.Bytes()); err != nil {
			return err
		}
		wires[i].L1.GetData(&ld)
		m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, r.messageSize(), ld[:])
		if err != nil {
			return err
		}
		m1p := mpint.Add(mpint.FromBytes(m1), k1)
		if err := r.io.SendData(m1p.Bytes()); err != nil {
			return err
		}
		if err := r.io.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Receive receives the wire labels with OT based on the flag values.
func (r *RSA) Receive(flags []bool, result []Label) error {
	for i := 0; i < len(flags); i++ {
		k, err := rand.Int(rand.Reader, r.pub.N)
		if err != nil {
			return err
		}
		// Receive random messages.
		var xb *big.Int
		if flags[i] {
			_, err = r.io.ReceiveData()
			if err != nil {
				return err
			}
			xb, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
		} else {
			xb, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
			_, err = r.io.ReceiveData()
			if err != nil {
				return err
			}
		}

		// Create and send V.
		e := big.NewInt(int64(r.pub.E))
		v := mpint.Mod(mpint.Add(xb, mpint.Exp(k, e, r.pub.N)), r.pub.N)
		if err := r.io.SendData(v.Bytes()); err != nil {
			return err
		}
		if err := r.io.Flush(); err != nil {
			return err
		}

		// Receive transfer messages.
		var mbp *big.Int
		if flags[i] {
			_, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
			mbp, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
		} else {
			mbp, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
			_, err = ReceiveBigInt(r.io)
			if err != nil {
				return err
			}
		}

		mbBytes := make([]byte, r.messageSize())
		mbIntBytes := mpint.Sub(mbp, k).Bytes()
		ofs := len(mbBytes) - len(mbIntBytes)
		copy(mbBytes[ofs:], mbIntBytes)

		mb, err := pkcs1.ParseEncryptionBlock(mbBytes)
		if err != nil {
			return err
		}
		result[i].SetBytes(mb)
	}
	return nil
}
