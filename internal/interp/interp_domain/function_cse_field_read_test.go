// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

//go:build !nocse

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCseStructFieldReadGeneralBank(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[0].op)
	require.Equal(t, opLoadIntConst, cf.body[1].op)
	require.Equal(t, opMoveGeneral, cf.body[2].op)
	require.Equal(t, uint8(6), cf.body[2].a, "matched read keeps its dest reg")
	require.Equal(t, uint8(5), cf.body[2].b, "MOVE_GENERAL src is first-read dest")
}

func TestCseStructFieldReadScalarTier0Int(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldIntT0, 3, 4, 7),
		mk(opLoadIntConst, 9, 0, 0),
		mk(opGetStructFieldIntT0, 8, 4, 7),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldIntT0, cf.body[0].op)
	require.Equal(t, opDrillTier1, cf.body[2].op)
	require.Equal(t, uint8(subOpMoveInt), cf.body[2].a)
	require.Equal(t, uint8(8), cf.body[2].b, "MOVE_INT dest preserves matched read dest")
	require.Equal(t, uint8(3), cf.body[2].c, "MOVE_INT src is first-read dest")
}

func TestCseStructFieldReadScalarTier0Uint(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldUint, 2, 1, 5),
		mk(opGetStructFieldUint, 3, 1, 5),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opDrillTier1, cf.body[1].op)
	require.Equal(t, uint8(subOpMoveUint), cf.body[1].a)
	require.Equal(t, uint8(3), cf.body[1].b)
	require.Equal(t, uint8(2), cf.body[1].c)
}

func TestCseStructFieldReadScalarTier0Float(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldFloat, 0, 1, 5),
		mk(opGetStructFieldFloat, 2, 1, 5),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opDrillTier1, cf.body[1].op)
	require.Equal(t, uint8(subOpMoveFloat), cf.body[1].a)
}

func TestCseStructFieldReadScalarTier0Bool(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldBool, 0, 1, 5),
		mk(opGetStructFieldBool, 2, 1, 5),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opDrillTier1, cf.body[1].op)
	require.Equal(t, uint8(subOpMoveBool), cf.body[1].a)
}

func TestCseStructFieldReadTier1String(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opDrillTier1, uint8(subOpGetStructFieldString), 5, 4),
		mk(opExt, 10, 0, 0),
		mk(opLoadIntConst, 1, 0, 0),
		mk(opDrillTier1, uint8(subOpGetStructFieldString), 6, 4),
		mk(opExt, 10, 0, 0),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opDrillTier1, cf.body[0].op, "first read preserved")
	require.Equal(t, uint8(subOpGetStructFieldString), cf.body[0].a)
	require.Equal(t, opDrillTier1, cf.body[3].op)
	require.Equal(t, uint8(subOpMoveString), cf.body[3].a, "second read became MOVE_STRING")
	require.Equal(t, uint8(6), cf.body[3].b, "MOVE_STRING dest preserved")
	require.Equal(t, uint8(5), cf.body[3].c, "MOVE_STRING src is first-read dest")
	require.Equal(t, opNop, cf.body[4].op, "trailing EXT word nopped")
}

func TestCseStructFieldReadBailsOnInterveningCall(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opCall, 0, 0, 0),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[2].op, "second read preserved after call")
}

func TestCseStructFieldReadBailsOnInterveningSetDifferentField(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opSetStructFieldGeneral, 4, 7, 3),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[2].op, "second read preserved after set to a different field of same receiver (conservative)")
}

func TestCseStructFieldReadBailsOnReceiverWrite(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opMoveGeneral, 4, 7, 0),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[2].op, "second read preserved after receiver overwrite")
}

func TestCseStructFieldReadMovePropagation(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opMoveGeneral, 9, 5, 0),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opMoveGeneral, cf.body[2].op, "second read CSE'd via alias")
	require.Equal(t, uint8(6), cf.body[2].a)
	require.Equal(t, uint8(5), cf.body[2].b, "smaller-index alias chosen as source")
}

func TestCseStructFieldReadDifferentReceiverNotMatched(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opGetStructFieldGeneral, 6, 7, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op, "different receiver, not a match")
}

func TestCseStructFieldReadDifferentLayoutNotMatched(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opGetStructFieldGeneral, 6, 4, 3),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op, "different layout, not a match")
}

func TestCseStructFieldReadPostSetGetGeneralBank(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opSetStructFieldGeneral, 4, 9, 2),
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opMoveGeneral, cf.body[2].op, "GET after SET became MOVE")
	require.Equal(t, uint8(5), cf.body[2].a)
	require.Equal(t, uint8(9), cf.body[2].b, "MOVE_GENERAL source is SET's value register")
}

func TestCseStructFieldReadPostSetGetBailsOnMutation(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opSetStructFieldGeneral, 4, 9, 2),
		mk(opMoveGeneral, 9, 8, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[2].op, "value reg overwritten; GET preserved")
}

func TestCseStructFieldReadBailsAtJumpTarget(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(0)
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opJumpIfFalse, 0, lo, hi),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.elideRedundantStructFieldRead(context.Background(), cf.body)
	require.Equal(t, opGetStructFieldGeneral, cf.body[2].op, "PC 2 is a jump target of PC 1's JUMP_IF_FALSE; CSE refused")
}
