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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProvideAsmHandlerJumpTableEntries_ProducesNonEmpty(t *testing.T) {
	t.Parallel()
	entries := ProvideAsmHandlerJumpTableEntries()
	require.NotEmpty(t, entries, "the asmgen entry-point must produce at least one handler installation entry")
}

func TestProvideAsmHandlerJumpTableEntries_NamesAreUnique(t *testing.T) {
	t.Parallel()
	entries := ProvideAsmHandlerJumpTableEntries()

	seen := make(map[string]int, len(entries))
	for index, entry := range entries {
		require.NotEmpty(t, entry.Name, "entry %d has empty Name", index)
		_, duplicate := seen[entry.Name]
		require.False(t, duplicate,
			"entry %d duplicates name %q first seen at index %d", index, entry.Name, seen[entry.Name])
		seen[entry.Name] = index
	}
}

func TestProvideAsmHandlerJumpTableEntries_ShimEntriesAppended(t *testing.T) {
	t.Parallel()
	staticEntries := buildStaticJumpTableEntries()
	allEntries := ProvideAsmHandlerJumpTableEntries()

	require.GreaterOrEqual(t, len(allEntries), len(staticEntries),
		"appendTier2ShimEntries must not drop any static entry")
}

func TestBuildStaticJumpTableEntries_NonEmpty(t *testing.T) {
	t.Parallel()
	entries := buildStaticJumpTableEntries()
	require.NotEmpty(t, entries)
}

func TestTier0Builders_ProduceEntries(t *testing.T) {
	t.Parallel()

	tier0Cases := []struct {
		entries func() []asmHandlerJumpTableEntryStub
		name    string
	}{
		{name: "tier0AsmEntries", entries: wrapTier0AsmEntries},
		{name: "tier0LoadConstEntries", entries: wrapTier0LoadConstEntries},
		{name: "tier0IntArithEntries", entries: wrapTier0IntArithEntries},
		{name: "tier0FloatArithEntries", entries: wrapTier0FloatArithEntries},
		{name: "tier0UintArithEntries", entries: wrapTier0UintArithEntries},
		{name: "tier0ComparisonEntries", entries: wrapTier0ComparisonEntries},
		{name: "tier0StringEntries", entries: wrapTier0StringEntries},
		{name: "tier0ControlFlowEntries", entries: wrapTier0ControlFlowEntries},
	}

	for _, tc := range tier0Cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.entries()
			require.NotEmpty(t, got, "%s must produce at least one entry", tc.name)
		})
	}
}

func TestTier1Builders_ProduceEntries(t *testing.T) {
	t.Parallel()

	tier1Cases := []struct {
		entries func() []asmHandlerJumpTableEntryStub
		name    string
	}{
		{name: "tier1MoveAndControlEntries", entries: wrapTier1MoveAndControlEntries},
		{name: "tier1StructFieldEntries", entries: wrapTier1StructFieldEntries},
		{name: "tier1SliceTypedEntries", entries: wrapTier1SliceTypedEntries},
		{name: "tier1ComplexEntries", entries: wrapTier1ComplexEntries},
		{name: "tier1MathTrigEntries", entries: wrapTier1MathTrigEntries},
		{name: "tier1StrconvEntries", entries: wrapTier1StrconvEntries},
		{name: "tier1CapAndBoxEntries", entries: wrapTier1CapAndBoxEntries},
		{name: "tier1MakeSliceEntries", entries: wrapTier1MakeSliceEntries},
		{name: "tier1ArithUnaryEntries", entries: wrapTier1ArithUnaryEntries},
		{name: "tier1ConversionEntries", entries: wrapTier1ConversionEntries},
		{name: "tier1MathScalarEntries", entries: wrapTier1MathScalarEntries},
		{name: "tier1MiscEntries", entries: wrapTier1MiscEntries},
		{name: "tier2IncDecEntries", entries: wrapTier2IncDecEntries},
	}

	for _, tc := range tier1Cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.entries()
			require.NotEmpty(t, got, "%s must produce at least one entry", tc.name)
		})
	}
}

type asmHandlerJumpTableEntryStub struct{}

func wrapTier0AsmEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0AsmEntries()))
}

func wrapTier0LoadConstEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0LoadConstEntries()))
}

func wrapTier0IntArithEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0IntArithEntries()))
}

func wrapTier0FloatArithEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0FloatArithEntries()))
}

func wrapTier0UintArithEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0UintArithEntries()))
}

func wrapTier0ComparisonEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0ComparisonEntries()))
}

func wrapTier0StringEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0StringEntries()))
}

func wrapTier0ControlFlowEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier0ControlFlowEntries()))
}

func wrapTier1MoveAndControlEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1MoveAndControlEntries()))
}

func wrapTier1StructFieldEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1StructFieldEntries()))
}

func wrapTier1SliceTypedEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1SliceTypedEntries()))
}

func wrapTier1ComplexEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1ComplexEntries()))
}

func wrapTier1MathTrigEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1MathTrigEntries()))
}

func wrapTier1StrconvEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1StrconvEntries()))
}

func wrapTier1CapAndBoxEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1CapAndBoxEntries()))
}

func wrapTier1MakeSliceEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1MakeSliceEntries()))
}

func wrapTier1ArithUnaryEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1ArithUnaryEntries()))
}

func wrapTier1ConversionEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1ConversionEntries()))
}

func wrapTier1MathScalarEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1MathScalarEntries()))
}

func wrapTier1MiscEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier1MiscEntries()))
}

func wrapTier2IncDecEntries() []asmHandlerJumpTableEntryStub {
	return make([]asmHandlerJumpTableEntryStub, len(tier2IncDecEntries()))
}
