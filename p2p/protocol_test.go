//
// protocol_test.go
//
// Copyright (c) 2023-2025 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"fmt"
	"testing"
)

var tests = []interface{}{
	byte(42),
	uint16(43),
	uint32(44),
	"Hello, world!",
	make([]byte, 1024),
	make([]byte, 2*1024*1024),
	make([]byte, 64*1024*1024),
}

func writer(c *Conn) {
	for _, test := range tests {
		switch d := test.(type) {
		case byte:
			if err := c.SendByte(d); err != nil {
				fmt.Printf("SendByte: %v\n", err)
			}

		case uint16:
			if err := c.SendUint16(int(d)); err != nil {
				fmt.Printf("SendUint16: %v\n", err)
			}

		case uint32:
			if err := c.SendUint32(int(d)); err != nil {
				fmt.Printf("SendUint32: %v\n", err)
			}

		case string:
			if err := c.SendString(d); err != nil {
				fmt.Printf("SendString: %v\n", err)
			}

		case []byte:
			if err := c.SendData(d); err != nil {
				fmt.Printf("SendData [%v]byte: %v\n", len(d), err)
			}

		default:
			fmt.Printf("writer: invalid data: %v(%T)\n", test, test)
		}
	}
	if err := c.Flush(); err != nil {
		fmt.Printf("Flush: %v\n", err)
	}
}

func TestProtocol(t *testing.T) {
	cw, c := Pipe()

	go writer(cw)

	for _, test := range tests {
		switch d := test.(type) {
		case byte:
			v, err := c.ReceiveByte()
			if err != nil {
				t.Fatalf("ReceiveByte: %v", err)
			}
			if v != d {
				t.Errorf("ReceiveByte: got %v, expected %v", v, d)
			}

		case uint16:
			v, err := c.ReceiveUint16()
			if err != nil {
				t.Fatalf("ReceiveUint16: %v", err)
			}
			if v != int(d) {
				t.Errorf("ReceiveUint16: got %v, expected %v", v, d)
			}

		case uint32:
			v, err := c.ReceiveUint32()
			if err != nil {
				t.Fatalf("ReceiveUint32: %v", err)
			}
			if v != int(d) {
				t.Errorf("ReceiveUint32: got %v, expected %v", v, d)
			}

		case string:
			v, err := c.ReceiveString()
			if err != nil {
				t.Fatalf("ReceiveString: %v", err)
			}
			if v != d {
				t.Errorf("ReceiveString: got %v, expected %v", v, d)
			}

		case []byte:
			v, err := c.ReceiveData()
			if err != nil {
				t.Fatalf("ReceiveData: %v", err)
			}
			if len(v) != len(d) {
				t.Errorf("ReceiveData: got [%v]byte, expected [%v]byte",
					len(v), len(d))
			}

		default:
			t.Errorf("invalid value: %v(%T)", test, test)
		}
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
