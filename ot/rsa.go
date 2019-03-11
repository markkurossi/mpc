//
// rsa.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/ot/mpint"
	"github.com/markkurossi/mpc/pkcs1"
)

var (
	UnknownInput = errors.New("Unknown input")
)

func RandomData(size int) ([]byte, error) {
	m := make([]byte, size)
	_, err := rand.Read(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

type Wire struct {
	Label0 []byte
	Label1 []byte
}

type Inputs map[int]Wire

func (i Inputs) String() string {
	var result string

	for k, v := range i {
		str := fmt.Sprintf("%d={%x,%x}", k, v.Label0, v.Label1)
		if len(result) > 0 {
			result += ", "
		}
		result += str
	}
	return result
}

type Sender struct {
	key    *rsa.PrivateKey
	inputs Inputs
}

func NewSender(keyBits int, inputs Inputs) (*Sender, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	return &Sender{
		key:    key,
		inputs: inputs,
	}, nil
}

func (s *Sender) MessageSize() int {
	return s.key.PublicKey.Size()
}

func (s *Sender) PublicKey() *rsa.PublicKey {
	return &s.key.PublicKey
}

func (s *Sender) NewTransfer(input int) (*SenderXfer, error) {
	w, ok := s.inputs[input]
	if !ok {
		return nil, UnknownInput
	}
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
		input:  w,
		x0:     x0,
		x1:     x1,
	}, nil
}

type SenderXfer struct {
	sender *Sender
	input  Wire
	x0     []byte
	x1     []byte
	k0     *big.Int
	k1     *big.Int
}

func (s *SenderXfer) MessageSize() int {
	return s.sender.MessageSize()
}

func (s *SenderXfer) RandomMessages() ([]byte, []byte) {
	return s.x0, s.x1
}

func (s *SenderXfer) ReceiveV(data []byte) {
	v := mpint.FromBytes(data)
	x0 := mpint.FromBytes(s.x0)
	x1 := mpint.FromBytes(s.x1)

	s.k0 = mpint.Exp(mpint.Sub(v, x0), s.sender.key.D, s.sender.key.PublicKey.N)
	s.k1 = mpint.Exp(mpint.Sub(v, x1), s.sender.key.D, s.sender.key.PublicKey.N)
}

func (s *SenderXfer) Messages() ([]byte, []byte, error) {
	m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(),
		s.input.Label0)
	if err != nil {
		return nil, nil, err
	}
	m0p := mpint.Add(mpint.FromBytes(m0), s.k0)

	m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(),
		s.input.Label1)
	if err != nil {
		return nil, nil, err
	}
	m1p := mpint.Add(mpint.FromBytes(m1), s.k1)

	return m0p.Bytes(), m1p.Bytes(), nil
}

type Receiver struct {
	pub *rsa.PublicKey
}

func NewReceiver(pub *rsa.PublicKey) (*Receiver, error) {
	return &Receiver{
		pub: pub,
	}, nil
}

func (r *Receiver) MessageSize() int {
	return r.pub.Size()
}

func (r *Receiver) NewTransfer(bit int) (*ReceiverXfer, error) {
	return &ReceiverXfer{
		receiver: r,
		bit:      bit,
	}, nil
}

type ReceiverXfer struct {
	receiver *Receiver
	bit      int
	k        *big.Int
	v        *big.Int
	mb       []byte
}

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

func (r *ReceiverXfer) V() []byte {
	return r.v.Bytes()
}

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

func (r *ReceiverXfer) Message() (m []byte, bit int) {
	return r.mb, r.bit
}
