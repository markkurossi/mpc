//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"testing"

	"github.com/markkurossi/mpc/ot"
)

func TestFx(t *testing.T) {
	testFx(t, false)
	testFx(t, true)
}

func testFx(t *testing.T, b bool) {
	fp, tp := ot.NewPipe()

	ch := make(chan interface{})

	go fxReceiver(tp, ch, b)

	oti := ot.NewCO()
	err := oti.InitSender(fp)
	if err != nil {
		t.Fatal(err)
	}

	a, err := NewLabel()
	if err != nil {
		t.Fatal(err)
	}
	r, err := FxSend(oti, a)
	if err != nil {
		t.Fatal(err)
	}

	ret := <-ch
	switch xb := ret.(type) {
	case error:
		t.Fatal(xb)

	case Label:
		if b {
			// b = 1 implies that xb = x1 = r⊕a
			tst := r
			tst.Xor(a)
			if !tst.Equal(xb) {
				t.Errorf("b=1, got %v, expected %v\n", xb, tst)
			}
		} else {
			// b = 0 implies that xb = x0 = r
			if !xb.Equal(r) {
				t.Errorf("b=0: got %v, expected %v\n", xb, r)
			}
		}

	default:
		t.Fatalf("unexpected result: %v(%T)", ret, ret)
	}
}

func fxReceiver(pipe ot.IO, ch chan interface{}, bit bool) {
	defer close(ch)

	oti := ot.NewCO()
	err := oti.InitReceiver(pipe)
	if err != nil {
		ch <- err
		return
	}

	xb, err := FxReceive(oti, bit)
	if err != nil {
		ch <- err
		return
	}

	ch <- xb
}
