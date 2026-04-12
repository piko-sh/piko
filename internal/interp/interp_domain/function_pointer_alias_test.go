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

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAliasAnalysisAllocsAreDistinct(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
			mk(opAllocIndirect, 1, 0, 0),
			mk(opMoveGeneral, 2, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.NotNil(t, cf.aliasInfo)
	require.False(t, cf.aliasInfo.mayAlias(2, 0, 1),
		"two distinct opAllocIndirect produce different alias classes")
}

func TestAliasAnalysisSameRegisterAlwaysAliases(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.True(t, cf.aliasInfo.mayAlias(0, 0, 0),
		"a register always aliases itself")
}

func TestAliasAnalysisWildOnCall(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
			mk(opAllocIndirect, 1, 0, 0),
			mk(opCall, 0, 0, 0),
			mk(opMoveGeneral, 2, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.True(t, cf.aliasInfo.mayAlias(3, 0, 1),
		"call wildens all general registers; everything may-aliases everything")
}

func TestAliasAnalysisMovePropagatesClass(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
			mk(opMoveGeneral, 1, 0, 0),
			mk(opAllocIndirect, 2, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.True(t, cf.aliasInfo.mayAlias(2, 0, 1),
		"reg 1 received reg 0 via move; same class; may-aliases")
	require.False(t, cf.aliasInfo.mayAlias(2, 0, 2),
		"reg 2 is a separate allocation; distinct class")
}

func TestAliasAnalysisParametersAreDistinct(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		parameterKinds: []registerKind{registerGeneral, registerGeneral},
		body: []instruction{
			mk(opMoveGeneral, 2, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.False(t, cf.aliasInfo.mayAlias(0, 0, 1),
		"two general-bank parameters seeded with distinct classes")
}

func TestAliasAnalysisMergeAtJumpTargetWidensToWild(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(2)
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
			mk(opJumpIfFalse, 0, lo, hi),
			mk(opAllocIndirect, 1, 0, 0),
			mk(opMoveGeneral, 0, 1, 0),
			mk(opMoveGeneral, 2, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.True(t, cf.aliasInfo.mayAlias(4, 0, 1),
		"reg 0 has different classes on the two paths reaching PC 4; merge widens to wild")
}

func TestAliasAnalysisEmptyFunctionHandled(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.Nil(t, cf.aliasInfo, "empty function does not allocate alias info")
}

func TestMayAliasNilInfoReturnsTrue(t *testing.T) {
	t.Parallel()
	var info *pointerAliasInfo
	require.True(t, info.mayAlias(0, 1, 2),
		"nil info: conservative may-alias")
}

func TestMayAliasOutOfRangePCReturnsTrue(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opAllocIndirect, 0, 0, 0),
		},
	}
	_ = runPointerAliasAnalysis(context.Background(), cf)
	require.True(t, cf.aliasInfo.mayAlias(999, 0, 1),
		"PC outside body: conservative may-alias")
}
