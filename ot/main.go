//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/markkurossi/mpc/ot/mpint"
	"github.com/markkurossi/mpc/pkcs1"
)

const (
	KeyBits = 2048
)

func RandomMessage(size int) ([]byte, error) {
	m := make([]byte, size)
	_, err := rand.Read(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

type Alice struct {
	key *rsa.PrivateKey
	m0  []byte
	m1  []byte
	x0  []byte
	x1  []byte
	k0  *big.Int
	k1  *big.Int
}

func NewAlice() (*Alice, error) {
	key, err := rsa.GenerateKey(rand.Reader, KeyBits)
	if err != nil {
		return nil, err
	}

	m0 := []byte{'M', 's', 'g', '0'}
	m1 := []byte{'M', 's', 'g', '1'}

	x0, err := RandomMessage(KeyBits / 8)
	if err != nil {
		return nil, err
	}
	x1, err := RandomMessage(KeyBits / 8)
	if err != nil {
		return nil, err
	}

	fmt.Printf("m0: %x\n", m0)
	fmt.Printf("m1: %x\n", m0)

	return &Alice{
		key: key,
		m0:  m0,
		m1:  m1,
		x0:  x0,
		x1:  x1,
	}, nil
}

func (a *Alice) PublicKey() *rsa.PublicKey {
	return &a.key.PublicKey
}

func (a *Alice) RandomMessages() ([]byte, []byte) {
	return a.x0, a.x1
}

func (a *Alice) ReceiveV(v *big.Int) {
	x0 := mpint.FromBytes(a.x0)
	x1 := mpint.FromBytes(a.x1)

	a.k0 = mpint.Exp(mpint.Sub(v, x0), a.key.D, a.key.PublicKey.N)
	a.k1 = mpint.Exp(mpint.Sub(v, x1), a.key.D, a.key.PublicKey.N)

	fmt.Printf("k0: %x\n", a.k0.Bytes())
	fmt.Printf("k1: %x\n", a.k1.Bytes())
}

func (a *Alice) Messages() ([]byte, []byte, error) {
	m0, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, KeyBits/8, a.m0)
	if err != nil {
		return nil, nil, err
	}
	m0p := mpint.Add(mpint.FromBytes(m0), a.k0)

	m1, err := pkcs1.NewEncryptionBlock(pkcs1.BT1, KeyBits/8, a.m1)
	if err != nil {
		return nil, nil, err
	}
	m1p := mpint.Add(mpint.FromBytes(m1), a.k1)

	return m0p.Bytes(), m1p.Bytes(), nil
}

func compare(a, b []byte) bool {
	if len(a) == len(b) {
		return bytes.Compare(a, b) == 0
	}
	if len(a) < len(b) {
		zeros := len(b) - len(a)
		for i := 0; i < zeros; i++ {
			if b[0] != 0 {
				return false
			}
		}
		return bytes.Compare(a, b[zeros:]) == 0
	} else {
		zeros := len(a) - len(b)
		for i := 0; i < zeros; i++ {
			if a[0] != 0 {
				return false
			}
		}
		return bytes.Compare(a[zeros:], b) == 0
	}
}

func (a *Alice) VerifyM0(data []byte) bool {
	return compare(a.m0, data)
}

func (a *Alice) VerifyM1(data []byte) bool {
	return compare(a.m1, data)
}

type Bob struct {
	bit int
	pub *rsa.PublicKey
	k   *big.Int
	v   *big.Int
	mb  []byte
}

func NewBob() (*Bob, error) {
	return &Bob{
		bit: 0,
	}, nil
}

func (b *Bob) ReceivePublicKey(pub *rsa.PublicKey) {
	b.pub = pub
}

func (b *Bob) ReceiveRandomMessages(x0, x1 []byte) error {
	kbuf, err := RandomMessage(KeyBits / 8)
	if err != nil {
		return err
	}
	b.k = mpint.Mod(mpint.FromBytes(kbuf), b.pub.N)

	var xb *big.Int
	if b.bit == 0 {
		xb = mpint.FromBytes(x0)
	} else {
		xb = mpint.FromBytes(x1)
	}

	fmt.Printf("k : %x\n", b.k.Bytes())

	e := big.NewInt(int64(b.pub.E))

	b.v = mpint.Mod(mpint.Add(xb, mpint.Exp(b.k, e, b.pub.N)), b.pub.N)

	fmt.Printf("v : %x\n", b.v.Bytes())

	return nil
}

func (b *Bob) V() *big.Int {
	return b.v
}

func (b *Bob) ReceiveMessages(m0p, m1p []byte, err error) error {
	if err != nil {
		return err
	}
	var mbp *big.Int
	if b.bit == 0 {
		mbp = mpint.FromBytes(m0p)
	} else {
		mbp = mpint.FromBytes(m1p)
	}
	mbBytes := make([]byte, KeyBits/8)
	mbIntBytes := mpint.Sub(mbp, b.k).Bytes()
	ofs := len(mbBytes) - len(mbIntBytes)
	copy(mbBytes[ofs:], mbIntBytes)

	mb, err := pkcs1.ParseEncryptionBlock(mbBytes)
	if err != nil {
		return err
	}
	b.mb = mb

	fmt.Printf("m%d: %x\n", b.bit, b.mb)
	return nil
}

func (b *Bob) Message() (m []byte, bit int) {
	return b.mb, b.bit
}

func main() {
	alice, err := NewAlice()
	if err != nil {
		log.Fatal(err)
	}

	bob, err := NewBob()
	if err != nil {
		log.Fatal(err)
	}

	bob.ReceivePublicKey(alice.PublicKey())
	err = bob.ReceiveRandomMessages(alice.RandomMessages())
	if err != nil {
		log.Fatal(err)
	}

	alice.ReceiveV(bob.V())
	err = bob.ReceiveMessages(alice.Messages())
	if err != nil {
		log.Fatal(err)
	}

	m, bit := bob.Message()
	var ret bool
	if bit == 0 {
		ret = alice.VerifyM0(m)
	} else {
		ret = alice.VerifyM1(m)
	}
	if !ret {
		fmt.Printf("Verify failed!\n")
		os.Exit(1)
	}
}
