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

type Sender struct {
	key    *rsa.PrivateKey
	inputs Inputs
	x0     []byte
	x1     []byte
	k0     *big.Int
	k1     *big.Int
}

func NewSender(keyBits int, inputs Inputs) (*Sender, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	sender := &Sender{
		key:    key,
		inputs: inputs,
	}

	x0, err := RandomData(sender.MessageSize())
	if err != nil {
		return nil, err
	}
	x1, err := RandomData(sender.MessageSize())
	if err != nil {
		return nil, err
	}

	sender.x0 = x0
	sender.x1 = x1

	return sender, nil
}

func (s *Sender) MessageSize() int {
	return s.key.PublicKey.Size()
}

func (s *Sender) PublicKey() *rsa.PublicKey {
	return &s.key.PublicKey
}

func (s *Sender) RandomMessages() ([]byte, []byte) {
	return s.x0, s.x1
}

func (s *Sender) ReceiveV(data []byte) {
	v := mpint.FromBytes(data)
	x0 := mpint.FromBytes(s.x0)
	x1 := mpint.FromBytes(s.x1)

	s.k0 = mpint.Exp(mpint.Sub(v, x0), s.key.D, s.key.PublicKey.N)
	s.k1 = mpint.Exp(mpint.Sub(v, x1), s.key.D, s.key.PublicKey.N)
}

func (s *Sender) Messages(input int) ([]byte, []byte, error) {
	w, ok := s.inputs[input]
	if !ok {
		return nil, nil, UnknownInput
	}

	m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(), w.Label0)
	if err != nil {
		return nil, nil, err
	}
	m0p := mpint.Add(mpint.FromBytes(m0), s.k0)

	m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, s.MessageSize(), w.Label1)
	if err != nil {
		return nil, nil, err
	}
	m1p := mpint.Add(mpint.FromBytes(m1), s.k1)

	return m0p.Bytes(), m1p.Bytes(), nil
}

type Receiver struct {
	bit int
	pub *rsa.PublicKey
	k   *big.Int
	v   *big.Int
	mb  []byte
}

func NewReceiver() (*Receiver, error) {
	return &Receiver{
		bit: 0,
	}, nil
}

func (r *Receiver) MessageSize() int {
	return r.pub.Size()
}

func (r *Receiver) ReceivePublicKey(pub *rsa.PublicKey) {
	r.pub = pub
}

func (r *Receiver) ReceiveRandomMessages(x0, x1 []byte) error {
	k, err := rand.Int(rand.Reader, r.pub.N)
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

	e := big.NewInt(int64(r.pub.E))
	r.v = mpint.Mod(mpint.Add(xb, mpint.Exp(r.k, e, r.pub.N)), r.pub.N)

	return nil
}

func (r *Receiver) V() []byte {
	return r.v.Bytes()
}

func (r *Receiver) ReceiveMessages(m0p, m1p []byte, err error) error {
	if err != nil {
		return err
	}
	var mbp *big.Int
	if r.bit == 0 {
		mbp = mpint.FromBytes(m0p)
	} else {
		mbp = mpint.FromBytes(m1p)
	}
	mbBytes := make([]byte, r.MessageSize())
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

func (r *Receiver) Message() (m []byte, bit int) {
	return r.mb, r.bit
}
