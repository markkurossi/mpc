//
// rsa_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"testing"
)

func benchmark(b *testing.B, keySize int) {
	m0 := []byte{'M', 's', 'g', '0'}
	m1 := []byte{'1', 'g', 's', 'M'}

	sender, err := NewSender(keySize, map[int]Wire{
		0: Wire{
			Label0: m0,
			Label1: m1,
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	receiver, err := NewReceiver(sender.PublicKey())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sXfer, err := sender.NewTransfer(0)
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
			ret = bytes.Compare(m0, m)
		} else {
			ret = bytes.Compare(m1, m)
		}
		if ret != 0 {
			b.Fatal("Verify failed!\n")
		}
	}
}

func BenchmarkOT512(b *testing.B) {
	benchmark(b, 512)
}

func BenchmarkOT1024(b *testing.B) {
	benchmark(b, 1024)
}

func BenchmarkOT2048(b *testing.B) {
	benchmark(b, 2048)
}
