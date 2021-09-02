//
// enc_test.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"testing"

	"github.com/markkurossi/mpc/ot"
)

func TestEnc(t *testing.T) {
	a, _ := ot.NewLabel(rand.Reader)
	b, _ := ot.NewLabel(rand.Reader)
	c, _ := ot.NewLabel(rand.Reader)
	tweak := uint32(42)
	var key [32]byte

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("Failed to create cipher: %s", err)
	}

	var data ot.LabelData

	encrypted := encrypt(cipher, a, b, c, tweak, &data)
	if err != nil {
		t.Fatalf("Encrypt failed: %s", err)
	}

	plain, err := decrypt(cipher, a, b, tweak, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %s", err)
	}

	if !c.Equal(plain) {
		t.Fatalf("Encrypt-decrypt failed")
	}
}

func BenchmarkEnc(b *testing.B) {
	var key [32]byte

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		b.Fatalf("Failed to create cipher: %s", err)
	}

	al, err := ot.NewLabel(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}
	bl, err := ot.NewLabel(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}
	cl, err := ot.NewLabel(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}

	var data ot.LabelData
	for i := 0; i < b.N; i++ {
		encrypt(cipher, al, bl, cl, uint32(i), &data)
	}
}

func BenchmarkLabelX(b *testing.B) {
	var key [32]byte

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		b.Fatalf("Failed to create cipher: %s", err)
	}

	al, err := NewLabelX(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}
	bl, err := NewLabelX(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}
	cl, err := NewLabelX(rand.Reader)
	if err != nil {
		b.Fatalf("Failed to create label: %s", err)
	}

	for i := 0; i < b.N; i++ {
		encryptX(cipher, al, bl, cl, uint32(i))
	}
}

type LabelX struct {
	d0 uint64
	d1 uint64
}

func NewLabelX(rand io.Reader) (LabelX, error) {
	var buf [16]byte
	var result LabelX

	_, err := rand.Read(buf[:])
	if err != nil {
		return result, err
	}
	result.SetData(&buf)

	return result, nil
}

func (l *LabelX) SetData(buf *[16]byte) {
	l.d0 = binary.BigEndian.Uint64((*buf)[0:8])
	l.d1 = binary.BigEndian.Uint64((*buf)[8:16])
}

func (l *LabelX) GetData(buf *[16]byte) {
	binary.BigEndian.PutUint64((*buf)[0:8], l.d0)
	binary.BigEndian.PutUint64((*buf)[8:16], l.d1)
}

func (l *LabelX) Xor(o LabelX) {
	l.d0 ^= o.d0
	l.d1 ^= o.d1
}

func (l *LabelX) Mul2() {
	l.d0 <<= 1
	l.d0 |= (l.d1 >> 63)
	l.d1 <<= 1
}

func (l *LabelX) Mul4() {
	l.d0 <<= 2
	l.d0 |= (l.d1 >> 62)
	l.d1 <<= 2
}

func NewTweakX(tweak uint32) LabelX {
	return LabelX{
		d1: uint64(tweak),
	}
}

func encryptX(alg cipher.Block, a, b, c LabelX, t uint32) LabelX {
	k := makeKX(a, b, t)

	var data [16]byte
	k.GetData(&data)
	alg.Encrypt(data[:], data[:])

	var pi LabelX
	pi.SetData(&data)

	pi.Xor(k)
	pi.Xor(c)

	return pi
}

func makeKX(a, b LabelX, t uint32) LabelX {
	a.Mul2()

	b.Mul4()
	a.Xor(b)

	a.Xor(NewTweakX(t))

	return a
}
