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

//go:build !nogvn

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGvnRewritesIdenticalAddInt(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opAddInt, 3, 1, 2),
		mk(opAddInt, 4, 1, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opDrillTier1, cf.body[3].op,
		"second opAddInt rewritten as tier-1 MOVE_INT")
	require.Equal(t, uint8(subOpMoveInt), cf.body[3].a)
	require.Equal(t, uint8(4), cf.body[3].b)
	require.Equal(t, uint8(3), cf.body[3].c)
}

func TestGvnRewritesCommutativeAddInt(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opAddInt, 3, 1, 2),
		mk(opAddInt, 4, 2, 1),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opDrillTier1, cf.body[3].op,
		"a+b and b+a have the same canonicalised key")
	require.Equal(t, uint8(subOpMoveInt), cf.body[3].a)
}

func TestGvnRefusesNonCommutativeReorder(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opSubInt, 3, 1, 2),
		mk(opSubInt, 4, 2, 1),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opSubInt, cf.body[3].op,
		"a-b and b-a are different values; no rewrite")
}

func TestGvnRewritesAddFloat(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadFloatConst, 1, 0, 0),
		mk(opLoadFloatConst, 2, 0, 0),
		mk(opAddFloat, 3, 1, 2),
		mk(opAddFloat, 4, 1, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opDrillTier1, cf.body[3].op)
	require.Equal(t, uint8(subOpMoveFloat), cf.body[3].a)
}

func TestGvnRewritesAddIntConst(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opAddIntConst, 2, 1, 5),
		mk(opAddIntConst, 3, 1, 5),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opDrillTier1, cf.body[2].op,
		"same operand + same constant index = same value")
}

func TestGvnRefusesDifferentConstants(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opAddIntConst, 2, 1, 5),
		mk(opAddIntConst, 3, 1, 6),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opAddIntConst, cf.body[2].op,
		"different const indices → different values")
}

func TestGvnRefusesAfterIntermediateWrite(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opAddInt, 3, 1, 2),
		mk(opLoadIntConst, 3, 0, 0),
		mk(opAddInt, 4, 1, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opAddInt, cf.body[4].op,
		"register 3 (candidate's dest) was overwritten; no rewrite")
}

func TestGvnRefusesAfterCall(t *testing.T) {
	t.Parallel()
	body := []instruction{
		mk(opLoadIntConst, 1, 0, 0),
		mk(opLoadIntConst, 2, 0, 0),
		mk(opAddInt, 3, 1, 2),
		mk(opCall, 0, 0, 0),
		mk(opAddInt, 4, 1, 2),
	}
	cf := &CompiledFunction{body: body}
	_ = cf.runFunctionGvn(context.Background())
	require.Equal(t, opAddInt, cf.body[4].op,
		"call clears value table; no rewrite")
}

func TestGvnRefusesEmptyFunction(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{}
	_ = cf.runFunctionGvn(context.Background())
	require.Empty(t, cf.body)
}
