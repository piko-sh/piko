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
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChargeArenaAllocation_DeltaWithinHeadroom(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)

	a.chargeArenaAllocation(256, 0)
	require.Equal(t, uint64(256), arenaTotalAllocatedBytes(a))

	a.chargeArenaAllocation(512, 256)
	require.Equal(t, uint64(512), arenaTotalAllocatedBytes(a),
		"delta charge should land at newCap, not newCap+oldCap")

	a.chargeArenaAllocation(1024, 512)
	require.Equal(t, uint64(1024), arenaTotalAllocatedBytes(a),
		"three doublings should charge final cap, not 2N-1")
}

func TestChargeArenaAllocation_TriggersMinorGCWithOwnerVM(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	a := newRegisterArena()
	vm.arena = a
	setArenaOwnerVM(t, a, vm)
	setArenaBudget(t, a, 4096)

	a.oldByteSlabs = [][]byte{make([]byte, 4000), make([]byte, 4000)}
	a.totalAllocatedBytes = 8000
	startCount := a.gcCount

	a.chargeArenaAllocation(1500, 0)

	require.Greater(t, a.gcCount, startCount,
		"chargeArenaAllocation must invoke MinorGC when over budget")
	require.LessOrEqual(t, arenaTotalAllocatedBytes(a), uint64(4096),
		"counter should be within budget after MinorGC reclaims dead slabs")
}

func TestChargeArenaAllocation_PanicsAfterMinorGCFails(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	a := newRegisterArena()
	vm.arena = a
	setArenaOwnerVM(t, a, vm)
	setArenaBudget(t, a, 1024)

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered, "expected panic when budget cannot be honoured")
		recoveredErr, ok := recovered.(error)
		require.True(t, ok, "expected error-typed panic, got %T", recovered)
		require.True(t, errors.Is(recoveredErr, errArenaBudgetExceeded),
			"panic should wrap errArenaBudgetExceeded, got %v", recoveredErr)
	}()

	a.chargeArenaAllocation(8192, 0)
}

func TestChargeArenaAllocation_PanicsWithoutOwnerVM(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	require.Nil(t, a.ownerVM, "test precondition")
	setArenaBudget(t, a, 1024)

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered, "expected panic when budget cannot be honoured")
		require.Equal(t, uint32(0), a.gcCount,
			"MinorGC must not run when ownerVM is nil")
	}()

	a.chargeArenaAllocation(8192, 0)
}

func TestChargeArenaAllocation_ShrinkSubtracts(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)

	a.chargeArenaAllocation(2048, 0)
	require.Equal(t, uint64(2048), arenaTotalAllocatedBytes(a))

	a.chargeArenaAllocation(1024, 2048)
	require.Equal(t, uint64(1024), arenaTotalAllocatedBytes(a),
		"shrink should subtract (oldCap - newCap)")
}

func TestChargeArenaAllocation_ShrinkClampsToZero(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)

	a.chargeArenaAllocation(512, 0)
	require.Equal(t, uint64(512), arenaTotalAllocatedBytes(a))

	a.chargeArenaAllocation(0, 4096)
	require.Equal(t, uint64(0), arenaTotalAllocatedBytes(a),
		"shrink with oldCap > counter must clamp to zero")
}

func TestChargeArenaAllocation_OverflowGuard(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, math.MaxUint64)
	a.totalAllocatedBytes = math.MaxUint64 - 100

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered, "expected overflow panic")
		recoveredErr, ok := recovered.(error)
		require.True(t, ok, "expected error-typed panic, got %T", recovered)
		require.True(t, errors.Is(recoveredErr, errArenaBudgetExceeded),
			"overflow guard must wrap errArenaBudgetExceeded")
	}()

	a.chargeArenaAllocation(1000, 0)
}

func TestChargeArenaAllocation_EqualCapsIsNoop(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)

	a.chargeArenaAllocation(1024, 0)
	require.Equal(t, uint64(1024), arenaTotalAllocatedBytes(a))

	a.chargeArenaAllocation(1024, 1024)
	require.Equal(t, uint64(1024), arenaTotalAllocatedBytes(a),
		"equal-caps charge must be a no-op")
}

func TestResetClearsBudgetCounter(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	setArenaBudget(t, a, 1<<20)

	a.chargeArenaAllocation(1024, 0)
	require.Equal(t, uint64(1024), arenaTotalAllocatedBytes(a))

	a.Reset()
	require.Equal(t, uint64(0), arenaTotalAllocatedBytes(a))
	require.Nil(t, a.ownerVM, "Reset should clear ownerVM")
}
