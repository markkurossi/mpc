//
// rsa.go
//
// Copyright (c) 2019-2022 Markku Rossi
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

// RandomData creates size bytes of random data.
func RandomData(size int) ([]byte, error) {
	m := make([]byte, size)
	_, err := rand.Read(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Label implements a 128 bit wire label.
type Label struct {
	D0 uint64
	D1 uint64
}

// LabelData contains lable data as byte array.
type LabelData [16]byte

func (l Label) String() string {
	return fmt.Sprintf("%016x%016x", l.D0, l.D1)
}

// Equal test if the labels are equal.
func (l Label) Equal(o Label) bool {
	return l.D0 == o.D0 && l.D1 == o.D1
}

// NewLabel creates a new random label.
func NewLabel(rand io.Reader) (Label, error) {
	var buf LabelData
	var label Label

	if _, err := rand.Read(buf[:]); err != nil {
		return label, err
	}
	label.SetData(&buf)
	return label, nil
}

// NewTweak creates a new label from the tweak value.
func NewTweak(tweak uint32) Label {
	return Label{
		D1: uint64(tweak),
	}
}

// S tests the label's S bit.
func (l Label) S() bool {
	return (l.D0 & 0x8000000000000000) != 0
}

// SetS sets the label's S bit.
func (l *Label) SetS(set bool) {
	if set {
		l.D0 |= 0x8000000000000000
	} else {
		l.D0 &= 0x7fffffffffffffff
	}
}

// Mul2 multiplies the label by 2.
func (l *Label) Mul2() {
	l.D0 <<= 1
	l.D0 |= (l.D1 >> 63)
	l.D1 <<= 1
}

// Mul4 multiplies the label by 4.
func (l *Label) Mul4() {
	l.D0 <<= 2
	l.D0 |= (l.D1 >> 62)
	l.D1 <<= 2
}

// Xor xors the label with the argument label.
func (l *Label) Xor(o Label) {
	l.D0 ^= o.D0
	l.D1 ^= o.D1
}

// GetData gets the labels as label data.
func (l Label) GetData(buf *LabelData) {
	binary.BigEndian.PutUint64(buf[0:8], l.D0)
	binary.BigEndian.PutUint64(buf[8:16], l.D1)
}

// SetData sets the labels from label data.
func (l *Label) SetData(data *LabelData) {
	l.D0 = binary.BigEndian.Uint64((*data)[0:8])
	l.D1 = binary.BigEndian.Uint64((*data)[8:16])
}

// Bytes returns the label data as bytes.
func (l Label) Bytes(buf *LabelData) []byte {
	l.GetData(buf)
	return buf[:]
}

// SetBytes sets the label data from bytes.
func (l *Label) SetBytes(data []byte) {
	l.D0 = binary.BigEndian.Uint64(data[0:8])
	l.D1 = binary.BigEndian.Uint64(data[8:16])
}

// Wire implements a wire with 0 and 1 labels.
type Wire struct {
	L0 Label
	L1 Label
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
