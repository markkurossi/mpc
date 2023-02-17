//
// pipe_test.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPipe(t *testing.T) {
	testData := []byte("Hello, world!")
	testInt := 42

	pipe, rPipe := NewPipe()
	done := make(chan error)

	go func(pipe *Pipe) {
		data, err := pipe.ReceiveData()
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		if bytes.Compare(data, testData) != 0 {
			done <- fmt.Errorf("ReceiveData: value mismatch: %x != %x",
				data, testData)
			pipe.Close()
			return
		}
		v, err := pipe.ReceiveUint32()
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		if v != testInt {
			done <- fmt.Errorf("ReceiveUint32: value mismatch")
			pipe.Close()
			return
		}
		_, err = pipe.ReceiveUint32()
		if err != io.EOF {
			done <- fmt.Errorf("expected EOF")
		}
		done <- nil
	}(rPipe)

	err := pipe.SendData(testData)
	if err != nil {
		t.Errorf("SendData failed: %v", err)
	}
	err = pipe.SendUint32(testInt)
	if err != nil {
		t.Errorf("SendUint32 failed: %v", err)
	}
	err = pipe.Close()
	if err != nil {
		t.Errorf("Close faield: %v", err)
	}

	err = <-done
	if err != nil {
		t.Errorf("consumer failed: %v", err)
	}
}
