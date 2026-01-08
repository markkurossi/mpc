//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

var mitccrhTests = []struct {
	key    string
	blocks []string
}{
	{
		key: "00000000000000000000000000000000",
		blocks: []string{
			"66e94bd4ef8a2c3b884cfa59ca342b2e",
			"f6b7bdd1caeebab574683893c4475484",
			"5c76002bc7206560efe550c80b8f12cc",
			"ec331f5dd1c5f40e28ea541caec913f6",
			"932c6dbf69255cf13edcdb72233acea3",
			"6d5c3e022e5a6f7be663b9e69bcea443",
			"e013d7f4fa7abd93a7b85db9cfff9b14",
			"f0a2a65d245dd6199dc70951c2478b65",
		},
	},
}

func TestMITCCRH(t *testing.T) {
	const (
		batchSize = 8
		k         = 8
		h         = 2
	)
	for _, test := range mitccrhTests {
		var s Label
		mitccrh := NewMITCCRH(s, batchSize)

		blks := make([]Label, k*h)
		mitccrh.Hash(blks, k, h)

		for i := 0; i < k; i++ {
			for j := 0; j < h; j++ {
				expected, err := hex.DecodeString(test.blocks[i])
				if err != nil {
					t.Fatal(err)
				}
				var result LabelData
				blks[i*h+j].GetData(&result)
				if !bytes.Equal(expected, result[:]) {
					fmt.Printf("%02d,%02d: %x != %x\n", i, j, expected, result)
				}
			}
		}
	}
}

func BenchmarkMITCCRH(b *testing.B) {
	const (
		batchSize = 8
		k         = 8
		h         = 2
	)
	var s Label
	mitccrh := NewMITCCRH(s, batchSize)

	var pad [2 * batchSize]Label

	for b.Loop() {
		mitccrh.Hash(pad[:], batchSize, 2)
	}
}
