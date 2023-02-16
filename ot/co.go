//
// co.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//
// Chou Orlandi OT - The Simplest Protocol for Oblivious Transfer.
//  - https://eprint.iacr.org/2015/267.pdf

package ot

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
	"math/big"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
	bo                   = binary.BigEndian
)

type COSender struct {
	curve elliptic.Curve
}

// NewCOSender creates a new CO OT sender. The Sender implements the
// following flow:
//
//	Sender				   Receiver
//	  |						 |
//	  |------- send A ------>|
//	  |						 |
//	  |<----- receive B -----|
//	  |						 |
//	  |-------send e{0,1}--->|
//	  |						 |
func NewCOSender() *COSender {
	return &COSender{
		curve: elliptic.P256(),
	}
}

func (s *COSender) Curve() elliptic.Curve {
	return s.curve
}

func (s *COSender) NewTransfer(m0, m1 []byte) (*COSenderXfer, error) {
	curveParams := s.curve.Params()

	// a <- Zp
	a, err := rand.Int(rand.Reader, curveParams.N)
	if err != nil {
		return nil, err
	}

	// A = G^a
	Ax, Ay := s.curve.ScalarBaseMult(a.Bytes())

	// Aa = A^a
	Aax, Aay := s.curve.ScalarMult(Ax, Ay, a.Bytes())

	// a:    {x,y}
	// a^-1: {x,-y}
	// AaInv = {Aax, -Aay}
	AaInvx := big.NewInt(0).Set(Aax)
	AaInvy := big.NewInt(0).Sub(curveParams.P, Aay)

	return &COSenderXfer{
		curve:  s.curve,
		hash:   sha256.New(),
		m0:     m0,
		m1:     m1,
		a:      a,
		Ax:     Ax,
		Ay:     Ay,
		AaInvx: AaInvx,
		AaInvy: AaInvy,
	}, nil
}

type COSenderXfer struct {
	curve  elliptic.Curve
	hash   hash.Hash
	m0     []byte
	m1     []byte
	a      *big.Int
	Ax     *big.Int
	Ay     *big.Int
	AaInvx *big.Int
	AaInvy *big.Int
	e0     []byte
	e1     []byte
}

func (s *COSenderXfer) A() (x, y []byte) {
	return s.Ax.Bytes(), s.Ay.Bytes()
}

func (s *COSenderXfer) ReceiveB(x, y []byte) {
	bx := big.NewInt(0).SetBytes(x)
	by := big.NewInt(0).SetBytes(y)

	bx, by = s.curve.ScalarMult(bx, by, s.a.Bytes())
	bax, bay := s.curve.Add(bx, by, s.AaInvx, s.AaInvy)

	s.e0 = xor(s.kdf(bx, by, 0), s.m0)
	s.e1 = xor(s.kdf(bax, bay, 0), s.m1)
}

func (s *COSenderXfer) E() (e0, e1 []byte) {
	return s.e0, s.e1
}

func (s *COSenderXfer) kdf(x, y *big.Int, id uint64) []byte {
	s.hash.Reset()
	s.hash.Write(x.Bytes())
	s.hash.Write(y.Bytes())

	var tmp [8]byte
	bo.PutUint64(tmp[:], id)
	s.hash.Write(tmp[:])

	return s.hash.Sum(nil)
}

func xor(a, b []byte) []byte {
	l := len(a)
	if len(b) < l {
		l = len(b)
	}
	for i := 0; i < l; i++ {
		a[i] ^= b[i]
	}
	return a[:l]
}

type COReceiver struct {
	curve elliptic.Curve
}

func NewCOReceiver(curve elliptic.Curve) *COReceiver {
	return &COReceiver{
		curve: curve,
	}
}

func (r *COReceiver) NewTransfer(bit uint) (*COReceiverXfer, error) {
	curveParams := r.curve.Params()

	// b <= Zp
	b, err := rand.Int(rand.Reader, curveParams.N)
	if err != nil {
		return nil, err
	}

	return &COReceiverXfer{
		curve: r.curve,
		hash:  sha256.New(),
		bit:   bit,
		b:     b,
	}, nil
}

type COReceiverXfer struct {
	curve elliptic.Curve
	hash  hash.Hash
	bit   uint
	b     *big.Int
	Bx    *big.Int
	By    *big.Int
	Asx   *big.Int
	Asy   *big.Int
}

func (r *COReceiverXfer) ReceiveA(x, y []byte) {
	Ax := big.NewInt(0).SetBytes(x)
	Ay := big.NewInt(0).SetBytes(y)

	Bx, By := r.curve.ScalarBaseMult(r.b.Bytes())
	if r.bit != 0 {
		Bx, By = r.curve.Add(Bx, By, Ax, Ay)
	}
	r.Bx = Bx
	r.By = By

	Asx, Asy := r.curve.ScalarMult(Ax, Ay, r.b.Bytes())
	r.Asx = Asx
	r.Asy = Asy
}

func (r *COReceiverXfer) B() (x, y []byte) {
	return r.Bx.Bytes(), r.By.Bytes()
}

func (r *COReceiverXfer) ReceiveE(e0, e1 []byte) []byte {
	var result []byte

	kdf := r.kdf(r.Asx, r.Asy, 0)

	if r.bit != 0 {
		result = xor(kdf, e1)
	} else {
		result = xor(kdf, e0)
	}
	return result
}

func (r *COReceiverXfer) kdf(x, y *big.Int, id uint64) []byte {
	r.hash.Reset()
	r.hash.Write(x.Bytes())
	r.hash.Write(y.Bytes())

	var tmp [8]byte
	bo.PutUint64(tmp[:], id)
	r.hash.Write(tmp[:])

	return r.hash.Sum(nil)
}
