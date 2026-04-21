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

// emitBoxToGeneral emits instructions to box a typed register value
// into a general (reflect.Value) register.
//
// Takes destGenReg (uint8) which is the destination general register.
// Takes source (varLocation) which is the source varLocation to box.
func (c *compiler) emitBoxToGeneral(_ context.Context, destGenReg uint8, source varLocation) {
	switch source.kind {
	case registerInt:
		c.function.emit(opMoveIntToGeneral, destGenReg, source.register, 0)
	case registerFloat:
		c.function.emit(opMoveFloatToGeneral, destGenReg, source.register, 0)
	case registerString:
		c.function.emit(opMoveStringToGeneral, destGenReg, source.register, 0)
	default:
		c.function.emit(opPackInterface, destGenReg, source.register, uint8(source.kind))
	}
}

// boxToGeneral boxes a typed varLocation into a general register
// using a persistent register allocation. No-op if already general.
//
// Takes location (*varLocation) which is the varLocation to box,
// updated in place.
func (c *compiler) boxToGeneral(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	genReg := c.scopes.alloc.alloc(registerGeneral)
	c.emitBoxToGeneral(ctx, genReg, *location)
	*location = varLocation{register: genReg, kind: registerGeneral}
}

// boxToGeneralTemp boxes a typed varLocation into a general register
// using a temporary register allocation. No-op if already general.
//
// Takes location (*varLocation) which is the varLocation to box,
// updated in place.
func (c *compiler) boxToGeneralTemp(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	genReg := c.scopes.alloc.allocTemp(registerGeneral)
	c.emitBoxToGeneral(ctx, genReg, *location)
	*location = varLocation{register: genReg, kind: registerGeneral}
}

// emitUnboxFromGeneral emits instructions to unbox a general register
// value into a typed register.
//
// Takes srcGenReg (uint8) which is the source general register to
// unbox.
// Takes destKind (registerKind) which is the target registerKind for
// the unboxed value.
//
// Returns varLocation of the unboxed value and any error.
func (c *compiler) emitUnboxFromGeneral(_ context.Context, srcGenReg uint8, destKind registerKind) (varLocation, error) {
	dest := c.scopes.alloc.alloc(destKind)
	switch destKind {
	case registerInt:
		c.function.emit(opMoveGeneralToInt, dest, srcGenReg, 0)
	case registerFloat:
		c.function.emit(opMoveGeneralToFloat, dest, srcGenReg, 0)
	case registerString:
		c.function.emit(opMoveGeneralToString, dest, srcGenReg, 0)
	default:
		c.function.emit(opUnpackInterface, dest, srcGenReg, uint8(destKind))
	}
	return varLocation{register: dest, kind: destKind}, nil
}
