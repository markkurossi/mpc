//
// rsa_test.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestCO(t *testing.T) {
	l0, _ := NewLabel(rand.Reader)
	l1, _ := NewLabel(rand.Reader)

	sender := NewCOSender()
	receiver := NewCOReceiver(sender.Curve())

	var l0Buf, l1Buf LabelData
	l0Data := l0.Bytes(&l0Buf)
	l1Data := l1.Bytes(&l1Buf)

	sXfer, err := sender.NewTransfer(l0Data, l1Data)
	if err != nil {
		t.Fatalf("COSender.NewTransfer: %v", err)
	}
	var bit uint = 1

	rXfer, err := receiver.NewTransfer(bit)
	if err != nil {
		t.Fatalf("COReceiver.NewTransfer: %v", err)
	}
	rXfer.ReceiveA(sXfer.A())
	sXfer.ReceiveB(rXfer.B())
	result := rXfer.ReceiveE(sXfer.E())

	var ret int
	if bit == 0 {
		ret = bytes.Compare(result, l0Data[:])
	} else {
		ret = bytes.Compare(result, l1Data[:])
	}
	if ret != 0 {
		t.Errorf("Verify failed")
	}
}

func BenchmarkCO(b *testing.B) {
	l0, _ := NewLabel(rand.Reader)
	l1, _ := NewLabel(rand.Reader)

	sender := NewCOSender()
	receiver := NewCOReceiver(sender.Curve())

	b.ResetTimer()

	var l0Buf, l1Buf LabelData
	for i := 0; i < b.N; i++ {
		l0Data := l0.Bytes(&l0Buf)
		l1Data := l1.Bytes(&l1Buf)
		sXfer, err := sender.NewTransfer(l0Data, l1Data)
		if err != nil {
			b.Fatalf("COSender.NewTransfer: %v", err)
		}
		bit := uint(i % 2)

		rXfer, err := receiver.NewTransfer(bit)
		if err != nil {
			b.Fatalf("COReceiver.NewTransfer: %v", err)
		}
		rXfer.ReceiveA(sXfer.A())
		sXfer.ReceiveB(rXfer.B())
		result := rXfer.ReceiveE(sXfer.E())

		var ret int
		if bit == 0 {
			ret = bytes.Compare(l0Data[:], result)
		} else {
			ret = bytes.Compare(l1Data[:], result)
		}
		if ret != 0 {
			b.Fatal("Verify failed")
		}
	}
}
