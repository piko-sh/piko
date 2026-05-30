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

// emitBoxToGeneral emits the instruction that boxes source into the general
// (reflect.Value) register destinationGeneralRegister, picking the dedicated
// MoveToGeneral sub-op for int, float, and string banks and the generic opPackInterface
// for others.
//
// Takes destinationGeneralRegister (uint8) which is the destination general register.
// Takes source (varLocation) which is the value being boxed.
func (c *compiler) emitBoxToGeneral(_ context.Context, destinationGeneralRegister uint8, source varLocation) {
	switch source.kind {
	case registerInt:
		c.function.emit(opDrillTier1, uint8(subOpMoveIntToGeneral), destinationGeneralRegister, source.register)
	case registerFloat:
		c.function.emit(opDrillTier1, uint8(subOpMoveFloatToGeneral), destinationGeneralRegister, source.register)
	case registerString:
		c.function.emit(opDrillTier1, uint8(subOpMoveStringToGeneral), destinationGeneralRegister, source.register)
	default:
		c.function.emit(opPackInterface, destinationGeneralRegister, source.register, uint8(source.kind))
	}
}

// emitTypedBox emits opPackTyped to box source into the general bank.
//
// Preserves source's exact source-level reflect.Type. When source carries no recorded
// type or when the type-table is exhausted the helper emits nothing and returns false,
// leaving the caller to fall back to the generic opPackInterface / move-to-general path.
//
// Takes destinationGeneralRegister (uint8) which receives the box.
// Takes source (varLocation) whose sourceType drives the boxing type.
//
// Returns bool which is true when opPackTyped (plus its opExt type-index word) was
// emitted.
func (c *compiler) emitTypedBox(destinationGeneralRegister uint8, source varLocation) bool {
	if source.sourceType == nil {
		return false
	}
	typeIndex, err := c.function.addTypeRef(source.sourceType)
	if err != nil {
		return false
	}
	c.function.emit(opPackTyped, destinationGeneralRegister, source.register, uint8(source.kind))
	c.function.emitExtension(typeIndex, 0)
	return true
}

// boxToGeneral boxes location into a freshly allocated persistent general register and
// updates location in place. No-op when location is already in the general bank.
//
// Takes location (*varLocation) which names the value to box and is rewritten in place to
// its new general-bank slot.
func (c *compiler) boxToGeneral(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	generalRegister := c.scopes.alloc.alloc(registerGeneral)
	c.emitBoxToGeneral(ctx, generalRegister, *location)
	*location = varLocation{register: generalRegister, kind: registerGeneral}
}

// boxToGeneralTemp boxes location into a freshly allocated temporary general register and
// updates location in place. No-op when location is already in the general bank.
//
// Takes location (*varLocation) which names the value to box and is rewritten in place to
// its new general-bank temporary slot.
func (c *compiler) boxToGeneralTemp(ctx context.Context, location *varLocation) {
	if location.kind == registerGeneral {
		return
	}
	generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.emitBoxToGeneral(ctx, generalRegister, *location)
	*location = varLocation{register: generalRegister, kind: registerGeneral}
}

// coerceToKind returns a varLocation in destinationKind, emitting the box or unbox
// instruction that bridges location.kind to destinationKind. Returns location unchanged
// when the banks already match.
//
// Takes location (varLocation) which is the value being coerced.
// Takes destinationKind (registerKind) which is the target bank.
//
// Returns the coerced varLocation; equals location when the kinds already match.
func (c *compiler) coerceToKind(ctx context.Context, location varLocation, destinationKind registerKind) varLocation {
	if location.kind == destinationKind {
		return location
	}
	if destinationKind == registerGeneral {
		destination := c.scopes.alloc.allocTemp(registerGeneral)
		c.emitBoxToGeneral(ctx, destination, location)
		return varLocation{register: destination, kind: registerGeneral}
	}
	if location.kind == registerGeneral {
		destination := c.scopes.alloc.allocTemp(destinationKind)
		switch destinationKind {
		case registerInt:
			c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToInt), destination, location.register)
		case registerFloat:
			c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToFloat), destination, location.register)
		case registerString:
			c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToString), destination, location.register)
		default:
			c.function.emit(opUnpackInterface, destination, location.register, uint8(destinationKind))
		}
		return varLocation{register: destination, kind: destinationKind}
	}
	if location.kind == registerInt && destinationKind == registerBool {
		destination := c.scopes.alloc.allocTemp(registerBool)
		c.function.emit(opDrillTier1, uint8(subOpIntToBool), destination, location.register)
		return varLocation{register: destination, kind: registerBool}
	}
	if location.kind == registerBool && destinationKind == registerInt {
		destination := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), destination, location.register)
		return varLocation{register: destination, kind: registerInt}
	}
	return location
}

// emitUnboxFromGeneral emits the instruction that unboxes the value in general register
// sourceGeneralRegister into a freshly allocated register of destinationKind. The
// returned error is always nil; the signature accommodates callers that need to propagate
// compilation errors.
//
// Takes sourceGeneralRegister (uint8) which is the source general register.
// Takes destinationKind (registerKind) which is the destination bank.
//
// Returns the freshly allocated destination location and a nil error.
func (c *compiler) emitUnboxFromGeneral(_ context.Context, sourceGeneralRegister uint8, destinationKind registerKind) (varLocation, error) {
	destination := c.scopes.alloc.alloc(destinationKind)
	switch destinationKind {
	case registerInt:
		c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToInt), destination, sourceGeneralRegister)
	case registerFloat:
		c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToFloat), destination, sourceGeneralRegister)
	case registerString:
		c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToString), destination, sourceGeneralRegister)
	default:
		c.function.emit(opUnpackInterface, destination, sourceGeneralRegister, uint8(destinationKind))
	}
	return varLocation{register: destination, kind: destinationKind}, nil
}
