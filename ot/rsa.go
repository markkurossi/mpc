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
	"encoding/binary"
	"fmt"
	"io"
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

type Label struct {
	d0 uint64
	d1 uint64
}

type LabelData [16]byte

func (l Label) String() string {
	return fmt.Sprintf("%016x%016x", l.d0, l.d1)
}

func (l Label) Undefined() bool {
	return l.d0 == 0 && l.d1 == 0
}

func (l Label) Equal(o Label) bool {
	return l.d0 == o.d0 && l.d1 == o.d1
}

func NewLabel(rand io.Reader) (Label, error) {
	var buf LabelData
	var label Label

	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return label, err
		}
		label.SetBytes(buf)
		if !label.Undefined() {
			return label, nil
		}
	}
}

func LabelFromData(data LabelData) Label {
	label := Label{}
	label.SetBytes(data)
	return label
}

func NewTweak(tweak uint32) Label {
	return Label{
		d1: uint64(tweak),
	}
}

func (l Label) S() bool {
	return (l.d0 & 0x8000000000000000) != 0
}

func (l *Label) SetS(set bool) {
	if set {
		l.d0 |= 0x8000000000000000
	} else {
		l.d0 &= 0x7fffffffffffffff
	}
}

func (l *Label) Mul2() {
	l.d0 <<= 1
	l.d0 |= (l.d1 >> 63)
	l.d1 <<= 1
}

func (l *Label) Mul4() {
	l.d0 <<= 2
	l.d0 |= (l.d1 >> 62)
	l.d1 <<= 2
}

func (l *Label) Xor(o Label) {
	l.d0 ^= o.d0
	l.d1 ^= o.d1
}

func (l Label) Data() LabelData {
	var result LabelData
	binary.BigEndian.PutUint64(result[0:8], l.d0)
	binary.BigEndian.PutUint64(result[8:16], l.d1)
	return result
}

func (l *Label) SetBytes(data LabelData) {
	l.d0 = binary.BigEndian.Uint64(data[0:8])
	l.d1 = binary.BigEndian.Uint64(data[8:16])
}

type Wire struct {
	L0 Label
	L1 Label
}

type Inputs map[int]Wire

func (i Inputs) String() string {
	var result string

	for k, v := range i {
		str := fmt.Sprintf("%d={%x,%x}", k, v.L0, v.L1)
		if len(result) > 0 {
			result += ", "
		}
		result += str
	}
	return result
}

type Sender struct {
	key *rsa.PrivateKey
}

func NewSender(keyBits int) (*Sender, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	return &Sender{
		key: key,
	}, nil
}

func (s *Sender) MessageSize() int {
	return s.key.PublicKey.Size()
}

func (s *Sender) PublicKey() *rsa.PublicKey {
	return &s.key.PublicKey
}

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

type SenderXfer struct {
	sender *Sender
	m0     []byte
	m1     []byte
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

func (r *Receiver) NewTransfer(bit uint) (*ReceiverXfer, error) {
	return &ReceiverXfer{
		receiver: r,
		bit:      bit,
	}, nil
}

type ReceiverXfer struct {
	receiver *Receiver
	bit      uint
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

func (r *ReceiverXfer) Message() (m []byte, bit uint) {
	return r.mb, r.bit
}
