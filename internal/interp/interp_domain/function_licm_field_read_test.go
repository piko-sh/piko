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

//go:build !nolicm

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func tier1Jump(offset int16) instruction {
	lo, hi := jumpOffset(offset)
	return mk(opDrillTier1, uint8(subOpJump), lo, hi)
}

func TestLicmHoistsLoopInvariantRead(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opLoadIntConst, 2, 0, 0),
		tier1Jump(-3),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, 5, len(cf.body), "body grew by 1 from hoist insertion")
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op, "hoist inserted at loop header")
	require.Equal(t, uint8(5), cf.body[1].a)
	require.Equal(t, uint8(4), cf.body[1].b)
	require.Equal(t, uint8(2), cf.body[1].c)
	require.Equal(t, opNop, cf.body[2].op, "original read became opNop")
}

func TestLicmRefusesWhenReceiverWrittenInLoop(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opMoveGeneral, 4, 7, 0),
		tier1Jump(-3),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "no hoist when receiver mutated in loop")
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op, "read unchanged")
}

func TestLicmRefusesWhenDestRewrittenInLoop(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opMoveGeneral, 5, 7, 0),
		tier1Jump(-3),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "no hoist when dest reg clobbered in loop")
}

func TestLicmRefusesWhenCallInLoop(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opCall, 0, 0, 0),
		tier1Jump(-3),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "no hoist when call in loop")
}

func TestLicmRefusesWhenSetSameFieldInLoop(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opSetStructFieldGeneral, 4, 9, 2),
		tier1Jump(-3),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "no hoist when same struct field written in loop")
}

func TestLicmIgnoresForwardJumpsAsNonLoops(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opGetStructFieldGeneral, 5, 4, 2),
		tier1Jump(3),
		mk(opGetStructFieldGeneral, 6, 4, 2),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "forward jump is not a loop; no hoist")
}

func TestLicmAdjustsForwardJumpAcrossInsert(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opJumpIfFalse, 0, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		tier1Jump(-2),
	}
	loEnd := body[0]
	_ = loEnd
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, 4, len(cf.body))
	require.Equal(t, opJumpIfFalse, cf.body[0].op)
	offset, isJump := jumpOffsetOf(cf.body[0])
	require.True(t, isJump)
	require.Equal(t, 1, offset, "forward jump's target shifted from PC1 to PC2; offset increments")
}

func TestLicmHoistsDominatingReadAfterConditional(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(1)
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opJumpIfTrue, 0, lo, hi),
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		tier1Jump(-4),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, 6, len(cf.body), "dominating read hoisted out of loop")
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op, "hoist inserted at pre-header (loop header)")
}

func TestLicmRefusesHoistWhenPreHeaderUnconditionallyJumps(t *testing.T) {
	t.Parallel()
	body := []instruction{
		tier1Jump(2),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opLoadIntConst, 1, 0, 0),
		tier1Jump(-2),
	}
	originalLen := len(body)
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, originalLen, len(cf.body), "no hoist when the slot preceding the loop header unconditionally jumps elsewhere")
	require.Equal(t, opGetStructFieldGeneral, cf.body[1].op)
}

func TestLicmPreservesBackEdgeTargetAfterShift(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opGetStructFieldGeneral, 5, 4, 2),
		mk(opLoadIntConst, 2, 0, 0),
		tier1Jump(-3),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.hoistLoopInvariantStructFieldReads(context.Background())
	require.Equal(t, 5, len(cf.body))
	backEdgePC := 4
	offset, isJump := jumpOffsetOf(cf.body[backEdgePC])
	require.True(t, isJump)
	require.Equal(t, -3, offset, "back-edge offset unchanged (both source and target shift)")
	require.Equal(t, backEdgePC+1+offset, 2, "back-edge lands at the opNop'd original read PC")
}
