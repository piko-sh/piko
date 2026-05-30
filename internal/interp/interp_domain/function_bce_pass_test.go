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

//go:build !nobce

package interp_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBceRewritesGetAfterLenLtJump(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body, intConstants: []int64{0}}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirectUnchecked, cf.body[4].op,
		"slice read inside proven-safe range becomes unchecked")
	require.Equal(t, uint8(4), cf.body[4].a)
	require.Equal(t, uint8(0), cf.body[4].b)
	require.Equal(t, uint8(2), cf.body[4].c)
}

func TestBceRewritesSetAfterLenLtJump(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceSetIntDirect, 0, 2, 4),
	}
	cf := &CompiledFunction{body: body, intConstants: []int64{0}}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceSetIntDirectUnchecked, cf.body[4].op,
		"slice write inside proven-safe range becomes unchecked")
}

func TestBceRefusesNegativeConstantIndex(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body, intConstants: []int64{-1}}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[4].op,
		"a negative index satisfies signed idx < len yet is out of range; "+
			"the unchecked rewrite must be refused")
}

func TestBceRefusesIndexWithUnknownSign(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[3].op,
		"index register with no proven lower bound must not be rewritten")
}

func TestBceRewritesGetWithSmallConstIndex(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opDrillTier1, uint8(subOpLoadIntConstSmall), 2, 3),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirectUnchecked, cf.body[4].op,
		"a small-const index is provably non-negative and may be rewritten")
}

func TestBceRefusesWhenLenRegOverwritten(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[5].op,
		"len register overwritten before lt-jump invalidates the fact")
}

func TestBceRefusesWhenIndexOverwrittenAfterFact(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[5].op,
		"index register overwritten after the proof invalidates the fact")
}

func TestBceRefusesAcrossCall(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opCall, 0, 0, 0),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[5].op,
		"call between proof and access invalidates the fact conservatively")
}

func TestBceRefusesAtJumpTarget(t *testing.T) {
	t.Parallel()
	jumpOver, _ := jumpOffset(0)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, jumpOver, 0),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[4].op,
		"access PC is a jump target; fact does not survive")
}

func TestBceDifferentSliceNoMatch(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 7, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[4].op,
		"fact was for slice register 0; access uses 7 - no rewrite")
}

func TestBceRewritesReflectGetAfterLenLtJump(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLen), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetInt, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body, intConstants: []int64{0}}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntUnchecked, cf.body[4].op,
		"reflect-bank slice read inside proven-safe range becomes unchecked")
	require.Equal(t, uint8(4), cf.body[4].a)
	require.Equal(t, uint8(0), cf.body[4].b)
	require.Equal(t, uint8(2), cf.body[4].c)
}

func TestBceRewritesReflectSetAfterLenLtJump(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLen), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceSetInt, 0, 2, 4),
	}
	cf := &CompiledFunction{body: body, intConstants: []int64{0}}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceSetIntUnchecked, cf.body[4].op,
		"reflect-bank slice write inside proven-safe range becomes unchecked")
}

func TestBceReflectBankDoesNotMatchTypedBankFact(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLenSliceIntDirect), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetInt, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetInt, cf.body[4].op,
		"typed-bank len fact must not authorise a reflect-bank rewrite even though the slice register number matches")
}

func TestBceTypedBankDoesNotMatchReflectBankFact(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLen), 1, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opLtInt, 3, 2, 1),
		mk(opJumpIfFalse, 3, lo, hi),
		mk(opSliceGetIntDirect, 4, 0, 2),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntDirect, cf.body[4].op,
		"reflect-bank len fact must not authorise a typed-bank rewrite even though the slice register number matches")
}

func TestBceStableDerefPreservesFactAcrossRedundantDeref(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLoadIntConstSmall), 0, 1),
		mk(opDeref, 1, 0, 0),
		mk(opDrillTier1, uint8(subOpLen), 1, 1),
		mk(opLtInt, 2, 0, 1),
		mk(opJumpIfFalse, 2, 4, 0),
		mk(opDeref, 1, 0, 0),
		mk(opSliceGetInt, 3, 1, 0),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetIntUnchecked, cf.body[6].op,
		"idempotent re-DEREF of the same pointer-to-slice must preserve the BCE proof so the access at PC 6 rewrites to unchecked")
}

func TestBceStableDerefDoesNotPreserveAcrossSourceWrite(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLoadIntConstSmall), 0, 1),
		mk(opDeref, 1, 0, 0),
		mk(opDrillTier1, uint8(subOpLen), 1, 1),
		mk(opLtInt, 2, 0, 1),
		mk(opJumpIfFalse, 2, 5, 0),
		mk(opDeref, 0, 1, 0),
		mk(opDeref, 1, 0, 0),
		mk(opSliceGetInt, 3, 1, 0),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetInt, cf.body[7].op,
		"a write to the DEREF source register between two DEREFs invalidates the idempotency - the value could differ")
}

func TestBceStableDerefDoesNotPreserveAcrossNonDerefWrite(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opDrillTier1, uint8(subOpLoadIntConstSmall), 0, 1),
		mk(opDeref, 1, 0, 0),
		mk(opDrillTier1, uint8(subOpLen), 1, 1),
		mk(opLtInt, 2, 0, 1),
		mk(opJumpIfFalse, 2, 5, 0),
		mk(opMoveGeneral, 1, 5, 0),
		mk(opDeref, 1, 0, 0),
		mk(opSliceGetInt, 3, 1, 0),
	}
	cf := &CompiledFunction{body: body}
	cf.elideRedundantBoundsChecks(cf.body)
	require.Equal(t, opSliceGetInt, cf.body[7].op,
		"a non-DEREF write to the destination register before the second DEREF should still allow re-establishment, but the current rule only matches against the most recent producer pattern so the rewrite remains conservative")
}
