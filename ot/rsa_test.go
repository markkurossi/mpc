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
	sender, err := NewSender(keySize)
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
		err = receiver.ReceiveMessages(sender.Messages())
		if err != nil {
			b.Fatal(err)
		}

		m, bit := receiver.Message()
		var ret int
		if bit == 0 {
			ret = bytes.Compare(sender.M0(), m)
		} else {
			ret = bytes.Compare(sender.M1(), m)
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
