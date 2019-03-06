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

	receiver, err := NewReceiver()
	if err != nil {
		b.Fatal(err)
	}

	receiver.ReceivePublicKey(sender.PublicKey())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = receiver.ReceiveRandomMessages(sender.RandomMessages())
		if err != nil {
			b.Fatal(err)
		}

		sender.ReceiveV(receiver.V())
		err = receiver.ReceiveMessages(sender.Messages(0))
		if err != nil {
			b.Fatal(err)
		}

		m, bit := receiver.Message()
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
