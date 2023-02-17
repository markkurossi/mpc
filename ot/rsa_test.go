//
// rsa_test.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func benchmark(b *testing.B, keySize int) {
	l0, _ := NewLabel(rand.Reader)
	l1, _ := NewLabel(rand.Reader)

	sender, err := NewSender(keySize)
	if err != nil {
		b.Fatal(err)
	}

	receiver, err := NewReceiver(sender.PublicKey())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	var l0Buf, l1Buf LabelData
	for i := 0; i < b.N; i++ {
		l0Data := l0.Bytes(&l0Buf)
		l1Data := l1.Bytes(&l1Buf)
		sXfer, err := sender.NewTransfer(l0Data, l1Data)
		if err != nil {
			b.Fatal(err)
		}
		rXfer, err := receiver.NewTransfer(1)
		if err != nil {
			b.Fatal(err)
		}
		err = rXfer.ReceiveRandomMessages(sXfer.RandomMessages())
		if err != nil {
			b.Fatal(err)
		}

		sXfer.ReceiveV(rXfer.V())
		err = rXfer.ReceiveMessages(sXfer.Messages())
		if err != nil {
			b.Fatal(err)
		}

		m, bit := rXfer.Message()
		var ret int
		if bit == 0 {
			ret = bytes.Compare(l0Data[:], m)
		} else {
			ret = bytes.Compare(l1Data[:], m)
		}
		if ret != 0 {
			b.Fatal("Verify failed!\n")
		}
	}
}

func BenchmarkRSA512(b *testing.B) {
	benchmark(b, 512)
}

func BenchmarkRSA1024(b *testing.B) {
	benchmark(b, 1024)
}

func BenchmarkRSA2048(b *testing.B) {
	benchmark(b, 2048)
}
