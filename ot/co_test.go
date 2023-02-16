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
	"fmt"
	"testing"
)

func TestCO(t *testing.T) {
	l0, _ := NewLabel(rand.Reader)
	l1, _ := NewLabel(rand.Reader)

	sender, err := NewCOSender()
	if err != nil {
		t.Fatalf("NewCOSender: %v", err)
	}

	receiver, err := NewCOReceiver(sender.CurveParams())
	if err != nil {
		t.Fatalf("NewCOReceiver: %v", err)
	}

	var l0Buf, l1Buf LabelData
	l0Data := l0.Bytes(&l0Buf)
	l1Data := l1.Bytes(&l1Buf)

	sXfer, err := sender.NewTransfer(l0Data, l1Data)
	if err != nil {
		t.Fatalf("COSender.NewTransfer: %v", err)
	}
	rXfer, err := receiver.NewTransfer(1)
	if err != nil {
		t.Fatalf("COReceiver.NewTransfer: %v", err)
	}
	err = rXfer.ReceiveA(sXfer.A())
	if err != nil {
		t.Fatalf("rXfer.ReceiveA: %v", err)
	}
	err = sXfer.ReceiveB(rXfer.B())
	if err != nil {
		t.Fatalf("sXfer.ReceiveB: %v", err)
	}
	result, err := rXfer.ReceiveE(sXfer.E())
	if err != nil {
		t.Fatalf("rXfer.ReceiveE: %v", err)
	}
	fmt.Printf("data0:  %x\n", l0Data)
	fmt.Printf("data1:  %x\n", l1Data)
	fmt.Printf("result: %x\n", result)
}

func BenchmarkCO(b *testing.B) {
	l0, _ := NewLabel(rand.Reader)
	l1, _ := NewLabel(rand.Reader)

	sender, err := NewCOSender()
	if err != nil {
		b.Fatalf("NewCOSender: %v", err)
	}

	receiver, err := NewCOReceiver(sender.CurveParams())
	if err != nil {
		b.Fatalf("NewCOReceiver: %v", err)
	}

	b.ResetTimer()

	var l0Buf, l1Buf LabelData
	for i := 0; i < b.N; i++ {
		l0Data := l0.Bytes(&l0Buf)
		l1Data := l1.Bytes(&l1Buf)
		sXfer, err := sender.NewTransfer(l0Data, l1Data)
		if err != nil {
			b.Fatalf("COSender.NewTransfer: %v", err)
		}
		var bit uint = 1

		rXfer, err := receiver.NewTransfer(bit)
		if err != nil {
			b.Fatalf("COReceiver.NewTransfer: %v", err)
		}
		err = rXfer.ReceiveA(sXfer.A())
		if err != nil {
			b.Fatalf("rXfer.ReceiveA: %v", err)
		}
		err = sXfer.ReceiveB(rXfer.B())
		if err != nil {
			b.Fatalf("sXfer.ReceiveB: %v", err)
		}
		result, err := rXfer.ReceiveE(sXfer.E())
		if err != nil {
			b.Fatalf("rXfer.ReceiveE: %v", err)
		}
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
