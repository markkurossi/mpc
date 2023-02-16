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
	"crypto/ecdsa"
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
	Priv *ecdsa.PrivateKey
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
func NewCOSender() (*COSender, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &COSender{
		Priv: priv,
	}, err
}

func (s *COSender) CurveParams() *elliptic.CurveParams {
	return s.Priv.Params()
}

func (s *COSender) NewTransfer(m0, m1 []byte) (*COSenderXfer, error) {
	curveParams := s.Priv.Params()

	// a <= Zp
	a, err := rand.Int(rand.Reader, curveParams.N)
	if err != nil {
		return nil, err
	}

	// A = G->mul_gen(a)
	//
	// Point res(this)
	// EC_POINT_mul(ec_group, res.point, a.n, Null, NUll, ctx)
	//   r = res.point
	//   n = a.n
	// int EC_POINT_mul(const EC_GROUP *group, EC_POINT *r, const BIGNUM *n,
	//	                const EC_POINT *q, const BIGNUM *m, BN_CTX *ctx);
	//
	//  => gen*n + q*m => r=gen*n

	Ax, Ay := curveParams.ScalarBaseMult(a.Bytes())
	Aax, Aay := curveParams.ScalarMult(Ax, Ay, a.Bytes())

	// BN_usub(point->y, group->field, point->y)
	// => result = group->field - point->y

	AaInvx := big.NewInt(0).Set(Aax)
	AaInvy := big.NewInt(0).Sub(curveParams.P, Aay)

	return &COSenderXfer{
		sender: s,
		hash:   sha256.New(),
		a:      a,
		m0:     m0,
		m1:     m1,
		Ax:     Ax,
		Ay:     Ay,
		AaInvx: AaInvx,
		AaInvy: AaInvy,
	}, nil
}

type COSenderXfer struct {
	sender *COSender
	hash   hash.Hash
	a      *big.Int
	m0     []byte
	m1     []byte
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

func (s *COSenderXfer) ReceiveB(x, y []byte) error {
	curveParams := s.sender.Priv.Params()

	bx := big.NewInt(0).SetBytes(x)
	by := big.NewInt(0).SetBytes(y)

	bx, by = curveParams.ScalarMult(bx, by, s.a.Bytes())
	bax, bay := curveParams.Add(bx, by, s.AaInvx, s.AaInvy)

	s.e0 = xor(s.kdf(bx, by, 0), s.m0)
	s.e1 = xor(s.kdf(bax, bay, 0), s.m1)

	return nil
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
	curveParams *elliptic.CurveParams
}

func NewCOReceiver(curveParams *elliptic.CurveParams) (*COReceiver, error) {
	return &COReceiver{
		curveParams: curveParams,
	}, nil
}

func (r *COReceiver) NewTransfer(bit uint) (*COReceiverXfer, error) {
	// b <= Zp
	b, err := rand.Int(rand.Reader, r.curveParams.N)
	if err != nil {
		return nil, err
	}

	return &COReceiverXfer{
		receiver:    r,
		curveParams: r.curveParams,
		hash:        sha256.New(),
		bit:         bit,
		b:           b,
	}, nil
}

type COReceiverXfer struct {
	receiver    *COReceiver
	curveParams *elliptic.CurveParams
	hash        hash.Hash
	bit         uint
	b           *big.Int
	Bx          *big.Int
	By          *big.Int
	Asx         *big.Int
	Asy         *big.Int
}

func (r *COReceiverXfer) ReceiveA(x, y []byte) error {
	Ax := big.NewInt(0).SetBytes(x)
	Ay := big.NewInt(0).SetBytes(y)

	Bx, By := r.curveParams.ScalarBaseMult(r.b.Bytes())
	if r.bit != 0 {
		Bx, By = r.curveParams.Add(Bx, By, Ax, Ay)
	}
	r.Bx = Bx
	r.By = By

	Asx, Asy := r.curveParams.ScalarMult(Ax, Ay, r.b.Bytes())
	r.Asx = Asx
	r.Asy = Asy

	return nil
}

func (r *COReceiverXfer) B() (x, y []byte) {
	return r.Bx.Bytes(), r.By.Bytes()
}

func (r *COReceiverXfer) ReceiveE(e0, e1 []byte) ([]byte, error) {
	var result []byte

	kdf := r.kdf(r.Asx, r.Asy, 0)

	if r.bit != 0 {
		result = xor(kdf, e1)
	} else {
		result = xor(kdf, e0)
	}
	return result, nil
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
