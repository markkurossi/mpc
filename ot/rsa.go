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
	"math/big"

	"github.com/markkurossi/mpc/ot/mpint"
	"github.com/markkurossi/mpc/pkcs1"
)

func RandomData(size int) ([]byte, error) {
	m := make([]byte, size)
	_, err := rand.Read(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

type Sender struct {
	key *rsa.PrivateKey
	m0  []byte
	m1  []byte
	x0  []byte
	x1  []byte
	k0  *big.Int
	k1  *big.Int
}

func NewSender(keyBits int) (*Sender, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	m0 := []byte{'M', 's', 'g', '0'}
	m1 := []byte{'1', 'g', 's', 'M'}

	sender := &Sender{
		key: key,
		m0:  m0,
		m1:  m1,
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

func (a *Sender) MessageSize() int {
	return a.key.PublicKey.Size()
}

func (a *Sender) PublicKey() *rsa.PublicKey {
	return &a.key.PublicKey
}

func (a *Sender) RandomMessages() ([]byte, []byte) {
	return a.x0, a.x1
}

func (a *Sender) ReceiveV(data []byte) {
	v := mpint.FromBytes(data)
	x0 := mpint.FromBytes(a.x0)
	x1 := mpint.FromBytes(a.x1)

	a.k0 = mpint.Exp(mpint.Sub(v, x0), a.key.D, a.key.PublicKey.N)
	a.k1 = mpint.Exp(mpint.Sub(v, x1), a.key.D, a.key.PublicKey.N)
}

func (a *Sender) Messages() ([]byte, []byte, error) {
	m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, a.MessageSize(), a.m0)
	if err != nil {
		return nil, nil, err
	}
	m0p := mpint.Add(mpint.FromBytes(m0), a.k0)

	m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, a.MessageSize(), a.m1)
	if err != nil {
		return nil, nil, err
	}
	m1p := mpint.Add(mpint.FromBytes(m1), a.k1)

	return m0p.Bytes(), m1p.Bytes(), nil
}

func (a *Sender) M0() []byte {
	return a.m0
}

func (a *Sender) M1() []byte {
	return a.m1
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

func (b *Receiver) MessageSize() int {
	return b.pub.Size()
}

func (b *Receiver) ReceivePublicKey(pub *rsa.PublicKey) {
	b.pub = pub
}

func (b *Receiver) ReceiveRandomMessages(x0, x1 []byte) error {
	k, err := rand.Int(rand.Reader, b.pub.N)
	if err != nil {
		return err
	}
	b.k = k

	var xb *big.Int
	if b.bit == 0 {
		xb = mpint.FromBytes(x0)
	} else {
		xb = mpint.FromBytes(x1)
	}

	e := big.NewInt(int64(b.pub.E))
	b.v = mpint.Mod(mpint.Add(xb, mpint.Exp(b.k, e, b.pub.N)), b.pub.N)

	return nil
}

func (b *Receiver) V() []byte {
	return b.v.Bytes()
}

func (b *Receiver) ReceiveMessages(m0p, m1p []byte, err error) error {
	if err != nil {
		return err
	}
	var mbp *big.Int
	if b.bit == 0 {
		mbp = mpint.FromBytes(m0p)
	} else {
		mbp = mpint.FromBytes(m1p)
	}
	mbBytes := make([]byte, b.MessageSize())
	mbIntBytes := mpint.Sub(mbp, b.k).Bytes()
	ofs := len(mbBytes) - len(mbIntBytes)
	copy(mbBytes[ofs:], mbIntBytes)

	mb, err := pkcs1.ParseEncryptionBlock(mbBytes)
	if err != nil {
		return err
	}
	b.mb = mb

	return nil
}

func (b *Receiver) Message() (m []byte, bit int) {
	return b.mb, b.bit
}
