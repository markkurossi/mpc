//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
)

func (prog *Program) StreamCircuit(params *utils.Params) error {
	stream, err := circuits.NewStreaming(params)
	if err != nil {
		return err
	}
	_ = stream
	return fmt.Errorf("StreamCircuit not implemented yet")
}
