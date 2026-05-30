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

func TestBytecodeVersionMajorIsSix(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint16(6), BytecodeVersionMajor,
		"BytecodeVersionMajor must be 6 after the struct-field unsafe-access sub-ops were added")
}

func TestBytecodeVersionMinorIsTwo(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint16(2), BytecodeVersionMinor,
		"BytecodeVersionMinor bumped to 2 when opSliceGetIntUnchecked / opSliceSetIntUnchecked were added for reflect-bank BCE")
}

func TestDrillMarkersAtIotaZero(t *testing.T) {
	t.Parallel()

	require.Equal(t, opcode(0), opDrillTier1,
		"opDrillTier1 must occupy main-iota 0 so the dispatch loop "+
			"can treat byte == 0 as the descent into tier 1")
	require.Equal(t, subOpcode(0), subOpDrillTier2,
		"subOpDrillTier2 must occupy tier-1 iota 0 so operand A == 0 "+
			"means descend into tier 2")
	require.Equal(t, subOpcodeTier2(0), subOpTier2DrillTier3,
		"subOpTier2DrillTier3 must occupy tier-2 iota 0 so operand B == 0 "+
			"means descend into tier 3")
	require.Equal(t, subOpcodeTier3(0), subOpTier3Nop,
		"subOpTier3Nop must occupy tier-3 iota 0 so the all-drill word "+
			"{0,0,0,0} encodes the no-op")
}

func TestOpNopAliasesDrillTier1(t *testing.T) {
	t.Parallel()

	require.Equal(t, opDrillTier1, opNop,
		"after the Phase 4 migration opNop is a const alias for "+
			"opDrillTier1 (both are opcode value 0); the tier-3 no-op "+
			"is the all-drill word {opDrillTier1, subOpDrillTier2, "+
			"subOpTier2DrillTier3, subOpTier3Nop} = {0,0,0,0} and is "+
			"fast-path-dispatched by the flat dispatch macro inline")
	require.Equal(t, opcode(0), opNop,
		"opNop must remain at opcode value 0 so existing "+
			"makeInstruction(opNop, 0, 0, 0) padding sites continue "+
			"emitting the all-zero word")
}
