//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"encoding/hex"
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
			"dc0ed85df9611abb7249cdd168c5467e",
			"c117d2238d53836acd92ddcdb85d6a21",
			"79c86d43f2be7fce99dd2c2133b0cf7c",
			"dbe01de67e346a800c4c4b4880311de4",
			"54ca53bb28791846e6b09a2757f014e4",
			"86495e4a9c80564982f41de01f2b9884",
			"d83636687394ca5538a73a2198ea4ab7",
		},
	},
}

func TestMITCCRH(t *testing.T) {
	const (
		batchSize = 8
		k         = 8
		h         = 2
	)
	for idx, test := range mitccrhTests {
		var s Block
		mitccrh := NewMITCCRH(s, batchSize)

		blks := make([]Block, k*h)
		mitccrh.Hash(blks, k, h)

		for i := 0; i < k; i++ {
			for j := 0; j < h; j++ {
				expected, err := hex.DecodeString(test.blocks[i])
				if err != nil {
					t.Fatal(err)
				}
				result := blks[i*h+j]
				if !bytes.Equal(expected, result[:]) {
					t.Errorf("test-%d: %02d,%02d: %x != %x\n",
						idx, i, j, expected, result)
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
	var s Block
	mitccrh := NewMITCCRH(s, batchSize)

	var pad [2 * batchSize]Block

	for b.Loop() {
		mitccrh.Hash(pad[:], batchSize, 2)
	}
}
