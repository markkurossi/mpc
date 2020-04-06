//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/compiler/utils"
)

type Streaming struct {
}

func NewStreaming(params *utils.Params) (*Streaming, error) {
	return &Streaming{}, nil
}
