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

func TestOperandShapeCriticalCoverage(t *testing.T) {
	t.Parallel()

	type op struct {
		opcode
		aRole   operandRole
		writesA bool
	}
	critical := []op{
		{opcode: opAddInt, aRole: roleRegInt, writesA: true},
		{opcode: opSubInt, aRole: roleRegInt, writesA: true},
		{opcode: opMulInt, aRole: roleRegInt, writesA: true},
		{opcode: opSliceGetInt, aRole: roleRegInt, writesA: true},
		{opcode: opSliceGetFloat, aRole: roleRegFloat, writesA: true},
		{opcode: opSliceGetString, aRole: roleRegString, writesA: true},
		{opcode: opSliceGetBool, aRole: roleRegBool, writesA: true},
		{opcode: opSliceGetUint, aRole: roleRegUint, writesA: true},
		{opcode: opGetFieldInt, aRole: roleRegInt, writesA: true},
		{opcode: opSetFieldInt, aRole: roleRegGeneral, writesA: false},
		{opcode: opPackInterface, aRole: roleRegGeneral, writesA: true},
		{opcode: opUnpackInterface, aRole: roleRegDynamic, writesA: true},
		{opcode: opLoadIntConst, aRole: roleRegInt, writesA: true},
		{opcode: opLoadFloatConst, aRole: roleRegFloat, writesA: true},
		{opcode: opLoadStringConst, aRole: roleRegString, writesA: true},
	}

	for _, tc := range critical {
		shape := operandShapes[tc.opcode]
		require.NotZero(t, shape.flags&shapeFlagDescribed,
			"opcode %s lacks shapeFlagDescribed", tc.opcode)
		require.Equal(t, tc.aRole, shape.a,
			"opcode %s expected role A %v, got %v", tc.opcode, tc.aRole, shape.a)
		require.Equal(t, tc.writesA, shape.writes[0],
			"opcode %s writes-A flag mismatch", tc.opcode)
	}
}

func TestOperandShapeReadKindsForTypedSliceGet(t *testing.T) {
	t.Parallel()

	cases := []opcode{
		opSliceGetInt,
		opSliceGetFloat,
		opSliceGetString,
		opSliceGetBool,
		opSliceGetUint,
	}
	for _, op := range cases {
		shape := operandShapes[op]
		require.Equal(t, roleRegGeneral, shape.b, "opcode %s operand B", op)
		require.True(t, shape.reads[1], "opcode %s reads operand B", op)
		require.Equal(t, roleRegInt, shape.c, "opcode %s operand C", op)
		require.True(t, shape.reads[2], "opcode %s reads operand C", op)
	}
}

func TestKindForRoleRoundTrip(t *testing.T) {
	t.Parallel()

	kinds := []registerKind{
		registerInt, registerFloat, registerString,
		registerBool, registerUint, registerComplex, registerGeneral,
	}
	for _, k := range kinds {
		role := roleForKind(k)
		got, ok := kindForRole(role)
		require.True(t, ok, "kindForRole(%v) should be register-shaped", role)
		require.Equal(t, k, got, "round-trip mismatch for %v", k)
	}
}

func TestKindForRoleNonRegisterRolesAreOpaque(t *testing.T) {
	t.Parallel()

	nonRegister := []operandRole{
		roleNone, roleConstIndex, roleFieldIndex, roleImmediate,
		roleTypeIndex, roleKindMarker, roleJumpOffsetLow, roleJumpOffsetHigh,
		roleCallSiteLow, roleCallSiteHigh, roleFollowsExtension, roleUnknown,
		roleRegDynamic,
	}
	for _, role := range nonRegister {
		_, ok := kindForRole(role)
		require.False(t, ok, "role %v should not be register-shaped", role)
	}
}
