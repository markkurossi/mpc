//
// ot_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"testing"
)

func benchmark(b *testing.B, keySize int) {
	alice, err := NewAlice(keySize)
	if err != nil {
		b.Fatal(err)
	}

	bob, err := NewBob()
	if err != nil {
		b.Fatal(err)
	}

	bob.ReceivePublicKey(alice.PublicKey())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = bob.ReceiveRandomMessages(alice.RandomMessages())
		if err != nil {
			b.Fatal(err)
		}

		alice.ReceiveV(bob.V())
		err = bob.ReceiveMessages(alice.Messages())
		if err != nil {
			b.Fatal(err)
		}

		m, bit := bob.Message()
		var ret bool
		if bit == 0 {
			ret = alice.VerifyM0(m)
		} else {
			ret = alice.VerifyM1(m)
		}
		if !ret {
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
