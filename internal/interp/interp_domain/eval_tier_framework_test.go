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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUmbrellaDisassemblyShowsSubOpName(t *testing.T) {
	t.Parallel()

	builder := newBytecodeBuilder()
	lengthConstantIndex := builder.addIntConst(3)
	builder.intRegisters(2).sliceIntRegisters(1).returnInt()
	builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
	builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 1)
	builder.emit(opExt, 1, 0, 0)
	builder.emit(opDrillTier1, uint8(subOpLenSliceIntDirect), 0, 0)
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()

	disassembly := compiled.Disassemble()

	require.Contains(t, disassembly, "1:MAKE_SLICE_INT",
		"expected disassembly to label the tier-1 sub-op by name; got:\n%s", disassembly)
	require.Contains(t, disassembly, "1:LEN_SLICE_INT_DIRECT",
		"expected disassembly to label the second tier-1 sub-op by name; got:\n%s", disassembly)
	require.False(t, strings.Contains(disassembly, "UMBRELLA"),
		"expected no legacy 'UMBRELLA' labels; got:\n%s", disassembly)
}

func TestUmbrellaTier2DispatchInvalid(t *testing.T) {
	t.Parallel()

	builder := newBytecodeBuilder()
	builder.intRegisters(1).returnInt()
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), 200, 0)
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()

	service := NewService()
	_, err := service.Execute(context.Background(), compiled)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tier-2 sub-op",
		"expected tier-2 dispatch error; got: %v", err)
}

func TestUmbrellaTier3DispatchNop(t *testing.T) {
	t.Parallel()

	builder := newBytecodeBuilder()
	builder.intRegisters(1).returnInt()
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), uint8(subOpTier3Nop))
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()

	service := NewService()
	_, err := service.Execute(context.Background(), compiled)
	require.NoError(t, err, "tier-3 nop must dispatch as a no-op")
}

func TestUmbrellaTier3DispatchInvalid(t *testing.T) {
	t.Parallel()

	builder := newBytecodeBuilder()
	builder.intRegisters(1).returnInt()
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), 200)
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()

	service := NewService()
	_, err := service.Execute(context.Background(), compiled)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tier-3 sub-op",
		"expected tier-3 dispatch error; got: %v", err)
}

func TestSubOpcodeStringerCoverage(t *testing.T) {
	t.Parallel()

	require.Equal(t, "MAKE_SLICE_INT", subOpMakeSliceInt.String())
	require.Equal(t, "DRILL_TIER2", subOpDrillTier2.String())
	require.Equal(t, "DRILL_TIER3", subOpTier2DrillTier3.String())
	require.Equal(t, "TIER3_NOP", subOpTier3Nop.String())
	require.Equal(t, "UNKNOWN_SUBOP", subOpcode(255).String())
	require.Equal(t, "UNKNOWN_TIER2", subOpcodeTier2(255).String())
	require.Equal(t, "UNKNOWN_TIER3", subOpcodeTier3(255).String())
}
