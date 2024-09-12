//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"slices"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/types"
)

func indexOfs(idx *Index, block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, *LRValue, types.Info, types.Info, types.Size, error) {
	undef := types.Undefined

	lv := idx

	var err error
	var values []ssa.Value
	var indices []arrayIndex
	var lrv *LRValue

	for lrv == nil {
		block, values, err = idx.Index.SSA(block, ctx, gen)
		if err != nil {
			return nil, nil, undef, undef, 0, err
		}
		if len(values) != 1 {
			return nil, nil, undef, undef, 0,
				ctx.Errorf(idx.Index, "invalid index")
		}
		index, err := values[0].ConstInt()
		if err != nil {
			return nil, nil, undef, undef, 0, ctx.Error(idx.Index, err.Error())
		}
		indices = append(indices, arrayIndex{
			i:   index,
			ast: idx.Index,
		})
		switch i := idx.Expr.(type) {
		case *Index:
			idx = i

		case *VariableRef:
			lrv, _, _, err = ctx.LookupVar(block, gen, block.Bindings, i)
			if err != nil {
				return nil, nil, undef, undef, 0, err
			}

		default:
			return nil, nil, undef, undef, 0, ctx.Errorf(idx.Expr,
				"invalid operation: cannot index %v (%T)",
				idx.Expr, idx.Expr)
		}
	}
	slices.Reverse(indices)

	lrv = lrv.Indirect()
	baseType := lrv.ValueType()
	elType := baseType
	var offset types.Size

	for _, index := range indices {
		if !elType.Type.Array() {
			return nil, nil, undef, undef, 0, ctx.Errorf(index.ast,
				"indexing non-array %s (%s)", lv.Expr, elType)
		}
		if index.i >= elType.ArraySize {
			return nil, nil, undef, undef, 0, ctx.Errorf(index.ast,
				"invalid array index %d (out of bounds for %d-element array)",
				index.i, elType.ArraySize)
		}
		offset += index.i * elType.ElementType.Bits
		elType = *elType.ElementType
	}

	return block, lrv, baseType, elType, offset, nil
}
