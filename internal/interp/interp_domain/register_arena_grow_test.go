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

func TestGrowByteSlab_ExpandsAndRetainsOld(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.byteSlab)
	originalRetained := len(a.oldByteSlabs)

	a.growByteSlab(originalCap*2 + 1)

	require.Greater(t, cap(a.byteSlab), originalCap)
	require.Equal(t, originalRetained+1, len(a.oldByteSlabs),
		"the old slab must be appended to the retention list")
	require.Equal(t, 0, a.byteIndex,
		"byteIndex must reset after grow so allocation resumes from zero")
}

func TestGrowIntBackingSlab_ExpandsAndCharges(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.intBackingSlab)
	beforeCharged := arenaTotalAllocatedBytes(a)

	a.growIntBackingSlab(originalCap*2 + 1)

	require.Greater(t, cap(a.intBackingSlab), originalCap)
	require.Greater(t, arenaTotalAllocatedBytes(a), beforeCharged,
		"grow must charge the delta to the budget counter")
	require.Equal(t, 0, a.intBackingIndex)
}

func TestGrowFloatBackingSlab_Expands(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.floatBackingSlab)

	a.growFloatBackingSlab(originalCap*2 + 1)
	require.Greater(t, cap(a.floatBackingSlab), originalCap)
}

func TestGrowStringBackingSlab_Expands(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.stringBackingSlab)

	a.growStringBackingSlab(originalCap*2 + 1)
	require.Greater(t, cap(a.stringBackingSlab), originalCap)
}

func TestGrowBoolBackingSlab_Expands(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.boolBackingSlab)

	a.growBoolBackingSlab(originalCap*2 + 1)
	require.Greater(t, cap(a.boolBackingSlab), originalCap)
}

func TestGrowUintBackingSlab_Expands(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)
	originalCap := cap(a.uintBackingSlab)

	a.growUintBackingSlab(originalCap*2 + 1)
	require.Greater(t, cap(a.uintBackingSlab), originalCap)
}

func TestGrowGenericBytesSlab_FirstCallAndRetention(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)

	a.genericBytesSlab = nil
	a.growGenericBytesSlab(initialGenericBytesCapacity)

	require.NotEmpty(t, a.genericBytesSlab,
		"genericBytesSlab must be initialised on first grow")
	require.Empty(t, a.oldGenericByteSlabs,
		"first grow of an empty slab should not retain anything")

	beforeCap := cap(a.genericBytesSlab)
	a.growGenericBytesSlab(beforeCap*2 + 1)
	require.Greater(t, cap(a.genericBytesSlab), beforeCap)
	require.Len(t, a.oldGenericByteSlabs, 1,
		"second grow must retain the previous slab")
}

func TestGrowSliceHeaderSlab_FirstCallAndRetention(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<30)

	a.sliceHeaderSlab = nil
	a.growSliceHeaderSlab()
	require.NotEmpty(t, a.sliceHeaderSlab)
	require.Empty(t, a.oldSliceHeaderSlabs,
		"first grow with empty slab must not retain")

	beforeCap := cap(a.sliceHeaderSlab)
	a.growSliceHeaderSlab()
	require.GreaterOrEqual(t, cap(a.sliceHeaderSlab), beforeCap)
	require.Len(t, a.oldSliceHeaderSlabs, 1)
}

func TestGrowStringBoxSlab(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	beforeCap := cap(a.stringBoxSlab)
	a.growStringBoxSlab()
	require.GreaterOrEqual(t, cap(a.stringBoxSlab), beforeCap)
}

func TestGrowIntBoxSlab(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	beforeCap := cap(a.intBoxSlab)
	a.growIntBoxSlab()
	require.GreaterOrEqual(t, cap(a.intBoxSlab), beforeCap)
}

func TestGrowFloatBoxSlab(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	beforeCap := cap(a.floatBoxSlab)
	a.growFloatBoxSlab()
	require.GreaterOrEqual(t, cap(a.floatBoxSlab), beforeCap)
}

func TestGrowUintBoxSlab(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	beforeCap := cap(a.uintBoxSlab)
	a.growUintBoxSlab()
	require.GreaterOrEqual(t, cap(a.uintBoxSlab), beforeCap)
}

func TestTrimOldByteSlabs(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()

	a.oldByteSlabs = make([][]byte, maxRetainedOldSlabsPerType+2)
	for index := range a.oldByteSlabs {
		a.oldByteSlabs[index] = make([]byte, 4)
	}
	a.trimOldByteSlabs()
	require.Len(t, a.oldByteSlabs, maxRetainedOldSlabsPerType+1,
		"trim should drop one entry from the head")

	a.oldByteSlabs = make([][]byte, maxRetainedOldSlabsPerType)
	a.trimOldByteSlabs()
	require.Len(t, a.oldByteSlabs, maxRetainedOldSlabsPerType,
		"trim should not act when at or under the cap")
}

func TestTrimOldGenericByteSlabs(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldGenericByteSlabs = make([][]byte, maxRetainedOldSlabsPerType+1)
	a.trimOldGenericByteSlabs()
	require.Len(t, a.oldGenericByteSlabs, maxRetainedOldSlabsPerType)
}

func TestTrimOldSliceHeaderSlabs(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldSliceHeaderSlabs = make([][]arenaSliceHeader, maxRetainedOldSlabsPerType+1)
	a.trimOldSliceHeaderSlabs()
	require.Len(t, a.oldSliceHeaderSlabs, maxRetainedOldSlabsPerType)
}

func TestTrimOldIntBackings(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldIntBackings = make([][]int64, maxRetainedOldSlabsPerType+1)
	a.trimOldIntBackings()
	require.Len(t, a.oldIntBackings, maxRetainedOldSlabsPerType)
}

func TestTrimOldFloatBackings(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldFloatBackings = make([][]float64, maxRetainedOldSlabsPerType+1)
	a.trimOldFloatBackings()
	require.Len(t, a.oldFloatBackings, maxRetainedOldSlabsPerType)
}

func TestTrimOldStringBackings(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldStringBackings = make([][]string, maxRetainedOldSlabsPerType+1)
	a.trimOldStringBackings()
	require.Len(t, a.oldStringBackings, maxRetainedOldSlabsPerType)
}

func TestTrimOldBoolBackings(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldBoolBackings = make([][]bool, maxRetainedOldSlabsPerType+1)
	a.trimOldBoolBackings()
	require.Len(t, a.oldBoolBackings, maxRetainedOldSlabsPerType)
}

func TestTrimOldUintBackings(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	a.oldUintBackings = make([][]uint64, maxRetainedOldSlabsPerType+1)
	a.trimOldUintBackings()
	require.Len(t, a.oldUintBackings, maxRetainedOldSlabsPerType)
}

func TestArenaBudgetLimit_HostAwareDefaultApplies(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	require.Zero(t, a.maxArenaBytes, "default arena should have no per-instance override")

	got := a.arenaBudgetLimit()
	require.Equal(t, defaultMaxArenaBytes, got,
		"zero override must select defaultMaxArenaBytes")
}

func TestArenaBudgetLimit_OverridePreserved(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)
	require.Equal(t, uint64(1<<20), a.arenaBudgetLimit())
}
