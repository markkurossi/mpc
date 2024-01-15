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
	testFx(t, 0, 0)
	testFx(t, 0, 1)
	testFx(t, 1, 0)
	testFx(t, 1, 1)
}

func testFx(t *testing.T, a, b uint) {
	fp, tp := ot.NewPipe()

	ch := make(chan interface{})

	go fxReceiver(tp, ch, b)

	oti := ot.NewCO()
	err := oti.InitSender(fp)
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

	case uint:
		// f×(a,b) = (c,d) where c⊕d = a·b
		if a*b != r^xb {
			t.Errorf("Fx: %v*%v=%v != %v⊕%v=%v\n", a, b, a*b, r, xb, r^xb)
		}
		if b == 1 {
			// b = 1 implies that xb = x1 = r⊕a
			tst := r ^ a
			if tst != xb {
				t.Errorf("b=1, got %v, expected %v\n", xb, tst)
			}
		} else {
			// b = 0 implies that xb = x0 = r
			if xb != r {
				t.Errorf("b=0: got %v, expected %v\n", xb, r)
			}
		}

	default:
		t.Fatalf("unexpected result: %v(%T)", ret, ret)
	}
}

func fxReceiver(pipe ot.IO, ch chan interface{}, b uint) {
	defer close(ch)

	oti := ot.NewCO()
	err := oti.InitReceiver(pipe)
	if err != nil {
		ch <- err
		return
	}

	xb, err := FxReceive(oti, b)
	if err != nil {
		ch <- err
		return
	}

	ch <- xb
}

func TestFxk(t *testing.T) {
	testFxk(t, 0)
	testFxk(t, 1)
}

func testFxk(t *testing.T, b uint) {
	fp, tp := ot.NewPipe()

	ch := make(chan interface{})

	go fxkReceiver(tp, ch, b)

	oti := ot.NewCO()
	err := oti.InitSender(fp)
	if err != nil {
		t.Fatal(err)
	}

	a, err := NewLabel()
	if err != nil {
		t.Fatal(err)
	}
	r, err := FxkSend(oti, a)
	if err != nil {
		t.Fatal(err)
	}

	ret := <-ch
	switch xb := ret.(type) {
	case error:
		t.Fatal(xb)

	case Label:
		// f×κ(s, b) = (c,d) where c⊕d = s·b and where s ∈ {0,1}κ and
		// b ∈ {0,1}.
		sb := a
		sb.Mul(b)
		cd := r
		cd.Xor(xb)
		if !sb.Equal(cd) {
			t.Errorf("Fx: %v*%v=%v != %v⊕%v=%v\n", a, b, sb, r, xb, cd)
		}
		if b == 1 {
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

func fxkReceiver(pipe ot.IO, ch chan interface{}, b uint) {
	defer close(ch)

	oti := ot.NewCO()
	err := oti.InitReceiver(pipe)
	if err != nil {
		ch <- err
		return
	}

	xb, err := FxkReceive(oti, b)
	if err != nil {
		ch <- err
		return
	}

	ch <- xb
}
