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
)

// emitTyped emits a three-operand opcode with automatic bank coercion.
//
// When any register-read operand arrives in a bank that disagrees with the opcode's
// operand-shape descriptor, an opMoveGeneralToX or opUnpackInterface coercion is inserted
// before the emit. Non-register operand roles (constants, immediates, jump offsets, kind
// markers, dynamic-bank slots) pass through untouched, and opcodes without an
// authoritative descriptor fall back to a direct emit, leaving operand validity to the
// caller.
//
// Takes ctx (context.Context) which is the compilation context, propagated through any
// coercions.
// Takes op (opcode) which is the opcode being emitted.
// Takes a (varLocation) which is the first operand location.
// Takes b (varLocation) which is the second operand location.
// Takes c2 (varLocation) which is the third operand location.
func (c *compiler) emitTyped(ctx context.Context, op opcode, a, b, c2 varLocation) {
	shape := operandShapes[op]
	if shape.flags&shapeFlagDescribed == 0 {
		c.function.emit(op, a.register, b.register, c2.register)
		return
	}
	a = c.coerceForOperand(ctx, shape.a, shape.reads[0], a)
	b = c.coerceForOperand(ctx, shape.b, shape.reads[1], b)
	c2 = c.coerceForOperand(ctx, shape.c, shape.reads[2], c2)
	c.function.emit(op, a.register, b.register, c2.register)
}

// coerceForOperand applies coerceToKind to a single operand position when role resolves
// to a concrete register bank and the source bank disagrees.
//
// Takes ctx (context.Context) which is the compilation context, propagated through the
// coercion.
// Takes role (operandRole) which is the operand role drawn from the shape descriptor.
// Takes reads (bool) which is true only for register-read positions; writes and
// non-register roles return location unchanged.
// Takes location (varLocation) which is the candidate operand location.
//
// Returns the original location when no coercion is required, otherwise the coerced
// replacement.
func (c *compiler) coerceForOperand(ctx context.Context, role operandRole, reads bool, location varLocation) varLocation {
	if !reads {
		return location
	}
	expectedKind, ok := kindForRole(role)
	if !ok {
		return location
	}
	if location.kind == expectedKind {
		return location
	}
	return c.coerceToKind(ctx, location, expectedKind)
}

// rawOperand wraps a non-register operand byte (a field index, immediate, kind marker,
// etc.) as a varLocation so it can flow through emitTyped uniformly.
//
// emitTyped never coerces non-register roles, so the kind only serves as a marker for the
// operand bank.
//
// Takes value (uint8) which is the raw operand byte.
//
// Returns a varLocation carrying value in the register field and registerGeneral as the
// kind marker.
func rawOperand(value uint8) varLocation {
	return varLocation{register: value, kind: registerGeneral}
}
