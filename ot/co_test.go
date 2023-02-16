//
// rsa_test.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
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
