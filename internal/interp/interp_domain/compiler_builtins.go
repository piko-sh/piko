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
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math/bits"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxMapSizeHintLog2 caps the log2 map-size hint encoded in subOpMakeMap's extension
	// word. Values above 31 cannot be encoded (1 << 31 exceeds the addressable map slot
	// space we model).
	maxMapSizeHintLog2 = 31
)

// compileBuiltinCall compiles a call to a built-in function.
//
// Takes name (string) which is the name of the built-in function to compile.
// Takes expression (*ast.CallExpr) which is the AST call expression.
//
// Returns varLocation holding the built-in call result and any compilation error.
func (c *compiler) compileBuiltinCall(ctx context.Context, name string, expression *ast.CallExpr) (varLocation, error) {
	switch name {
	case "len":
		return c.compileBuiltinLen(ctx, expression)
	case "append":
		return c.compileBuiltinAppend(ctx, expression)
	case "make":
		return c.compileBuiltinMake(ctx, expression)
	case "delete":
		return c.compileBuiltinDelete(ctx, expression)
	case "cap":
		return c.compileBuiltinCap(ctx, expression)
	case "copy":
		return c.compileBuiltinCopy(ctx, expression)
	case "new":
		return c.compileBuiltinNew(ctx, expression)
	case "panic", "recover", "close":
		return c.compileBuiltinFeatureGated(ctx, name, expression)
	case "print":
		return c.compileBuiltinPrint(ctx, expression, builtinPrint)
	case "println":
		return c.compileBuiltinPrint(ctx, expression, builtinPrintln)
	case "min":
		return c.compileBuiltinMinMax(ctx, expression, true)
	case "max":
		return c.compileBuiltinMinMax(ctx, expression, false)
	case "clear":
		return c.compileBuiltinClear(ctx, expression)
	case "real":
		return c.compileBuiltinReal(ctx, expression)
	case "imag":
		return c.compileBuiltinImag(ctx, expression)
	case "complex":
		return c.compileBuiltinComplex(ctx, expression)
	default:
		return varLocation{}, fmt.Errorf("unsupported built-in: %s at %s", name, c.positionString(expression.Pos()))
	}
}

// compileBuiltinFeatureGated compiles built-in calls that require a feature gate check
// before compilation (panic, recover, close).
//
// Takes name (string) which is the built-in function name.
// Takes expression (*ast.CallExpr) which is the AST call expression to compile.
//
// Returns varLocation holding the call result and any compilation error.
func (c *compiler) compileBuiltinFeatureGated(ctx context.Context, name string, expression *ast.CallExpr) (varLocation, error) {
	switch name {
	case "panic":
		if err := c.checkFeature(InterpFeaturePanicRecover, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinPanic(ctx, expression)
	case "recover":
		if err := c.checkFeature(InterpFeaturePanicRecover, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinRecover(ctx, expression)
	default:
		if err := c.checkFeature(InterpFeatureChannels, expression.Lparen); err != nil {
			return varLocation{}, err
		}
		return c.compileBuiltinClose(ctx, expression)
	}
}

// compileBuiltinLen compiles len(x).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the len call.
//
// Returns varLocation holding the length value and any compilation error.
func (c *compiler) compileBuiltinLen(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, ErrCompileBuiltinLenArgCount
	}

	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerInt)

	switch argumentLocation.kind {
	case registerString:
		c.function.emit(opDrillTier1, uint8(subOpLenString), dest, argumentLocation.register)
	case registerGeneral:
		c.function.emit(opDrillTier1, uint8(subOpLen), dest, argumentLocation.register)
	default:
		if directLenSubOp, ok := typedSliceDirectLenSubOp(argumentLocation.kind); ok {
			c.function.emit(opDrillTier1, uint8(directLenSubOp), dest, argumentLocation.register)
			return varLocation{register: dest, kind: registerInt}, nil
		}
		return varLocation{}, fmt.Errorf("len not supported for register kind %s", argumentLocation.kind)
	}

	return varLocation{register: dest, kind: registerInt}, nil
}

// compileBuiltinAppend compiles append(slice, elems...).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the append call.
//
// Returns varLocation holding the resulting slice and any compilation error.
func (c *compiler) compileBuiltinAppend(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) < 2 {
		return varLocation{}, ErrCompileBuiltinAppendArgCount
	}

	sliceLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	sliceType := c.info.Types[expression.Args[0]].Type
	typedAppendSubOp, typedAppendKind := c.pickTypedAppendVariant(sliceType)

	lastArgIndex := len(expression.Args) - 1
	for i := 1; i < len(expression.Args); i++ {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return varLocation{}, err
		}
		location = c.coerceEvalBoolResult(ctx, c.info, expression.Args[i], location)
		isSpread := i == lastArgIndex && expression.Ellipsis != token.NoPos
		sliceLocation = c.emitAppendStep(ctx, sliceLocation, location, isSpread, typedAppendSubOp, typedAppendKind, sliceType)
	}

	return sliceLocation, nil
}

// pickTypedAppendVariant selects the tier-1 typed-append sub-op (and the matching
// register kind) for the slice's element type, or returns (0, 0) when no typed variant
// applies.
//
// The five primitive-bank typed-append variants live in tier 1 as subOpAppend* sub-ops.
// Tier 0 carries the generic opAppend and opAppendSpread, reserving tier-0 opcode space
// for handlers that benefit from the faster dispatch path (ASM-accelerated handlers and
// ones used inside the inner ASM loop). Append trampolines to Go, so the extra tier-1
// dispatch cost is negligible.
//
// Takes sliceType (types.Type) which is the static type of the destination slice.
//
// Returns the tier-1 sub-opcode (0 when not eligible).
// Returns the element register kind matched by the sub-opcode.
func (c *compiler) pickTypedAppendVariant(sliceType types.Type) (subOpcode, registerKind) {
	if sliceType == nil {
		return 0, 0
	}
	sliceValue, ok := sliceType.Underlying().(*types.Slice)
	if !ok {
		return 0, 0
	}
	switch c.kindFor(sliceValue.Elem()) {
	case registerInt:
		return subOpAppendInt, registerInt
	case registerString:
		return subOpAppendString, registerString
	case registerFloat:
		return subOpAppendFloat, registerFloat
	case registerBool:
		return subOpAppendBool, registerBool
	case registerUint:
		return subOpAppendUint, registerUint
	default:
	}
	return 0, 0
}

// emitAppendStep emits the opcode for a single argument of append.
//
// Takes sliceLocation (varLocation) which is the current destination slice (chained
// across arguments).
// Takes location (varLocation) which is the compiled value to append.
// Takes isSpread (bool) which is true when the argument uses the `...` spread form.
// Takes typedAppendSubOp (subOpcode) selecting the tier-1 fast path, 0 to force the
// generic opAppend path.
// Takes typedAppendKind (registerKind) which must match location.kind for the typed path
// to fire.
// Takes sliceType (types.Type) used to detect the byte-slice specialisation.
//
// Returns the new sliceLocation after the emit.
func (c *compiler) emitAppendStep(ctx context.Context, sliceLocation, location varLocation, isSpread bool, typedAppendSubOp subOpcode, typedAppendKind registerKind, sliceType types.Type) varLocation {
	if isSpread {
		c.boxToGeneralTemp(ctx, &sliceLocation)
		c.boxToGeneralTemp(ctx, &location)
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opAppendSpread, dest, sliceLocation.register, location.register)
		return varLocation{register: dest, kind: registerGeneral}
	}
	if isTypedSliceKind(sliceLocation.kind) {
		if directOp, ok := typedDirectAppendSubOp(sliceLocation.kind); ok && location.kind == elementKindForTypedSlice(sliceLocation.kind) {
			return c.emitTypedSliceDirectAppend(sliceLocation, location, directOp)
		}
	}
	if typedAppendSubOp != 0 && location.kind == typedAppendKind {
		if dest, ok := c.tryEmitByteFastAppend(sliceLocation, location, typedAppendSubOp, sliceType); ok {
			return dest
		}
		return c.emitTypedAppend(sliceLocation, location, typedAppendSubOp)
	}
	c.boxToGeneralTemp(ctx, &location)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAppend, dest, sliceLocation.register, location.register)
	return varLocation{register: dest, kind: registerGeneral}
}

// emitTypedSliceDirectAppend emits a typed-bank-direct append. The destination slice
// lives on the same typed bank as the source, avoiding the reflect-boxing round trip the
// general-bank append pays.
//
// Encoding: opDrillTier1, A = direct sub-op id, B = dest reg, C = source slice reg.
// Extension word A = element reg.
//
// Takes sliceLocation (varLocation) which is the source slice.
// Takes location (varLocation) which is the element to append.
// Takes directOp (subOpcode) which is the typed-direct sub-op id.
//
// Returns the new typed-bank slice location.
func (c *compiler) emitTypedSliceDirectAppend(sliceLocation, location varLocation, directOp subOpcode) varLocation {
	dest := c.scopes.alloc.alloc(sliceLocation.kind)
	c.function.emit(opDrillTier1, uint8(directOp), dest, sliceLocation.register)
	c.function.emit(opExt, location.register, 0, 0)
	return varLocation{register: dest, kind: sliceLocation.kind}
}

// tryEmitByteFastAppend emits opAppendByteFast when the slice element is statically a
// byte.
//
// Skips the tier-1 sub-op dispatch entirely and emits a single tier-0 opcode that goes
// through a per-op direct exit (no processExitTier2 indirection) AND a hot-path Go
// handler that omits the reflect.TypeAssert cascade. Serves byte-builder patterns such as
// `*output = append(*output, '(')`.
//
// Takes sliceLocation (varLocation) which is the destination slice.
// Takes location (varLocation) which is the value to append.
// Takes typedAppendSubOp (subOpcode) which gates the path on subOpAppendUint.
// Takes sliceType (types.Type) used to confirm the element type is canonically a byte.
//
// Returns the new destination location and true on success; the zero value and false
// otherwise.
func (c *compiler) tryEmitByteFastAppend(sliceLocation, location varLocation, typedAppendSubOp subOpcode, sliceType types.Type) (varLocation, bool) {
	if typedAppendSubOp != subOpAppendUint || sliceType == nil {
		return varLocation{}, false
	}
	sliceValue, ok := sliceType.Underlying().(*types.Slice)
	if !ok {
		return varLocation{}, false
	}
	elemKind := sliceValue.Elem().Underlying().String()
	if elemKind != "byte" && elemKind != "uint8" {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAppendByteFast, dest, sliceLocation.register, location.register)
	return varLocation{register: dest, kind: registerGeneral}, true
}

// emitTypedAppend emits the tier-1 subOpAppend encoding.
//
// Encoding:
//
//	op = opDrillTier1, a = sub-op id, b = dest reg,
//	c = slice reg, extension.a = element reg.
//
// Takes sliceLocation (varLocation) which is the destination slice.
// Takes location (varLocation) which is the value to append.
// Takes typedAppendSubOp (subOpcode) which selects the typed variant.
//
// Returns the new destination location.
func (c *compiler) emitTypedAppend(sliceLocation, location varLocation, typedAppendSubOp subOpcode) varLocation {
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDrillTier1, uint8(typedAppendSubOp), dest, sliceLocation.register)
	c.function.emit(opExt, location.register, 0, 0)
	return varLocation{register: dest, kind: registerGeneral}
}

// compileBuiltinMake compiles make(type, arguments...).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the make call.
//
// Returns varLocation holding the newly created value and any compilation error.
func (c *compiler) compileBuiltinMake(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	tv, ok := c.info.Types[expression]
	if !ok || tv.Type == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for make at %s", errCompilation, c.positionString(expression.Pos()))
	}
	reflectType := c.typeToReflect(ctx, tv.Type)
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)

	switch reflectType.Kind() {
	case reflect.Slice:
		return c.compileMakeSlice(ctx, expression, dest, typeIndex)
	case reflect.Map:
		hint := c.extractMapSizeHint(expression)
		c.function.emit(opDrillTier1, uint8(subOpMakeMap), dest, 0)
		c.function.emitExtension(typeIndex, hint)
	case reflect.Chan:
		return c.compileMakeChannel(ctx, expression, dest, typeIndex)
	default:
		return varLocation{}, fmt.Errorf("make not supported for type %v at %s", reflectType, c.positionString(expression.Pos()))
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// extractMapSizeHint reads the second argument of make(mapType, hint) when it is a
// constant non-negative integer, and returns its encoded log2 hint (0 when no constant
// hint is present).
//
// Takes expression (*ast.CallExpr) which is the make call.
//
// Returns the encoded log2 hint clamped to [0, maxMapSizeHintLog2].
func (c *compiler) extractMapSizeHint(expression *ast.CallExpr) uint8 {
	if len(expression.Args) < 2 {
		return 0
	}
	tvHint, ok := c.info.Types[expression.Args[1]]
	if !ok || tvHint.Value == nil {
		return 0
	}
	hn, exact := constant.Int64Val(tvHint.Value)
	if !exact || hn <= 0 {
		return 0
	}
	return mapSizeHintLog2(int(hn))
}

// compileMakeSliceTyped emits bytecode for typed-bank make([]T, len[, cap]).
//
// Used when T resolves to a typed-storage bank: the destination register is allocated
// from the typed slice bank (e.g. registerSliceInt) and the allocation is dispatched
// through the umbrella opcode so that typed make ops do not consume direct opcode slots.
// compileBuiltinMake routes general slice allocations through the general-bank path so
// copy/append/range/native boundary code (which expects reflect.Value storage) keeps
// working unchanged.
//
// Takes expression (*ast.CallExpr) which is the AST call expression supplying the length
// and (optional) capacity arguments.
// Takes dest (uint8) which is the destination register in the typed slice bank chosen by
// the caller.
// Takes destinationBank (registerKind) which names the typed slice bank receiving the new
// slice header.
//
// Returns varLocation holding the typed slice and any compilation error.
//
//nolint:unused // gated typed-slice emission helper
func (c *compiler) compileMakeSliceTyped(ctx context.Context, expression *ast.CallExpr, dest uint8, destinationBank registerKind) (varLocation, error) {
	var lengthLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		lengthLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return varLocation{}, err
		}
	}
	capacityLocation := lengthLocation
	if len(expression.Args) >= makeSliceMinCapArgs {
		var err error
		capacityLocation, err = c.compileExpression(ctx, expression.Args[2])
		if err != nil {
			return varLocation{}, err
		}
	}
	subOp, ok := typedMakeSliceSubOp(destinationBank)
	if !ok {
		return varLocation{}, fmt.Errorf("compileMakeSliceTyped: unsupported bank %d at %s", destinationBank, c.positionString(expression.Pos()))
	}
	c.function.emit(opDrillTier1, uint8(subOp), dest, lengthLocation.register)
	c.function.emit(opExt, capacityLocation.register, 0, 0)
	return varLocation{register: dest, kind: destinationBank}, nil
}

// compileMakeSlice emits bytecode for make([]T, len[, cap]).
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing the make
// arguments.
// Takes dest (uint8) which is the destination general register for the new slice.
// Takes typeIndex (uint16) which is the type reference index for the slice type.
//
// Returns varLocation holding the new slice and any compilation error.
func (c *compiler) compileMakeSlice(ctx context.Context, expression *ast.CallExpr, dest uint8, typeIndex uint16) (varLocation, error) {
	var lenLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		lenLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return varLocation{}, err
		}
	}
	capLocation := lenLocation
	if len(expression.Args) >= makeSliceMinCapArgs {
		var err error
		capLocation, err = c.compileExpression(ctx, expression.Args[2])
		if err != nil {
			return varLocation{}, err
		}
	}
	c.function.emit(opMakeSlice, dest, lenLocation.register, capLocation.register)
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMakeChannel emits bytecode for make(chan T[, size]).
//
// Takes expression (*ast.CallExpr) which is the AST call expression containing the make
// arguments.
// Takes dest (uint8) which is the destination general register for the new channel.
// Takes typeIndex (uint16) which is the type reference index for the channel type.
//
// Returns varLocation holding the new channel and any compilation error.
func (c *compiler) compileMakeChannel(ctx context.Context, expression *ast.CallExpr, dest uint8, typeIndex uint16) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, expression.Lparen); err != nil {
		return varLocation{}, err
	}
	var sizeLocation varLocation
	if len(expression.Args) >= 2 {
		var err error
		sizeLocation, err = c.compileExpression(ctx, expression.Args[1])
		if err != nil {
			return varLocation{}, err
		}
	} else {
		sizeLocation.register = c.scopes.alloc.alloc(registerInt)
		sizeLocation.kind = registerInt
		constIndex, err := c.function.addIntConstant(0)
		if err != nil {
			return varLocation{}, err
		}
		c.function.emitWide(opLoadIntConst, sizeLocation.register, constIndex)
	}
	c.function.emit(opDrillTier1, uint8(subOpMakeChannel), dest, sizeLocation.register)
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinDelete compiles delete(map, key).
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the delete call.
//
// Returns an empty varLocation and any compilation error.
func (c *compiler) compileBuiltinDelete(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, ErrCompileBuiltinDeleteArgCount
	}

	mapLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	keyLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}

	c.boxToGeneral(ctx, &keyLocation)

	c.function.emit(opDrillTier1, uint8(subOpMapDelete), mapLocation.register, keyLocation.register)
	return varLocation{}, nil
}

// compileBuiltinReal compiles the built-in real() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the real call.
//
// Returns varLocation holding the extracted real component and any compilation error.
func (c *compiler) compileBuiltinReal(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileComplexExtract(ctx, expression, "real", subOpRealComplex)
}

// compileBuiltinImag compiles the built-in imag() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the imag call.
//
// Returns varLocation holding the extracted imaginary component and any compilation
// error.
func (c *compiler) compileBuiltinImag(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileComplexExtract(ctx, expression, "imag", subOpImagComplex)
}

// compileComplexExtract compiles a complex number component extraction (real or imag) via
// the umbrella opcode's sub-op dispatch.
//
// Takes expression (*ast.CallExpr) which is the AST call expression.
// Takes name (string) which is the builtin function name for error messages.
// Takes subOp (subOpcode) which selects the umbrella sub-handler (subOpRealComplex or
// subOpImagComplex).
//
// Returns varLocation holding the extracted float component and any compilation error.
func (c *compiler) compileComplexExtract(ctx context.Context, expression *ast.CallExpr, name string, subOp subOpcode) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, fmt.Errorf("%s requires exactly 1 argument", name)
	}
	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	if argumentLocation.kind != registerComplex {
		return varLocation{}, fmt.Errorf("%s requires a complex argument", name)
	}
	dest := c.scopes.alloc.alloc(registerFloat)
	c.function.emit(opDrillTier1, uint8(subOp), dest, argumentLocation.register)
	return varLocation{register: dest, kind: registerFloat}, nil
}

// compileBuiltinComplex compiles the built-in complex() function call.
//
// Takes expression (*ast.CallExpr) which is the AST call expression for the complex call.
//
// Returns varLocation holding the constructed complex value and any compilation error.
func (c *compiler) compileBuiltinComplex(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, ErrCompileBuiltinComplexArgCount
	}
	realLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	imagLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}
	if realLocation.kind != registerFloat {
		return varLocation{}, ErrCompileBuiltinComplexRequiresFloat
	}
	if imagLocation.kind != registerFloat {
		return varLocation{}, ErrCompileBuiltinComplexRequiresFloat
	}
	dest := c.scopes.alloc.alloc(registerComplex)
	c.function.emit(opBuildComplex, dest, realLocation.register, imagLocation.register)
	return varLocation{register: dest, kind: registerComplex}, nil
}

// compileBuiltinCap compiles cap(x).
//
// Takes expression (*ast.CallExpr) which is the call expression containing the argument.
//
// Returns the capacity value location and any compilation error.
func (c *compiler) compileBuiltinCap(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, ErrCompileBuiltinCapArgCount
	}

	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerInt)

	if argumentLocation.kind == registerGeneral {
		c.function.emit(opDrillTier1, uint8(subOpCap), dest, argumentLocation.register)
		return varLocation{register: dest, kind: registerInt}, nil
	}

	if directCapSubOp, ok := typedSliceDirectCapSubOp(argumentLocation.kind); ok {
		c.function.emit(opDrillTier1, uint8(directCapSubOp), dest, argumentLocation.register)
		return varLocation{register: dest, kind: registerInt}, nil
	}

	return varLocation{}, fmt.Errorf("cap not supported for register kind %s", argumentLocation.kind)
}

// compileBuiltinCopy compiles copy(dst, src).
//
// Takes expression (*ast.CallExpr) which is the call expression containing the arguments.
//
// Returns the number of elements copied as an int location and any compilation error.
func (c *compiler) compileBuiltinCopy(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, ErrCompileBuiltinCopyArgCount
	}

	dstLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	sourceLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}

	if subOp, ok := typedSliceDirectCopySubOp(dstLocation.kind); ok && dstLocation.kind == sourceLocation.kind {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOp), dstLocation.register, sourceLocation.register)
		c.function.emit(opExt, dest, 0, 0)
		return varLocation{register: dest, kind: registerInt}, nil
	}

	c.boxToGeneralTemp(ctx, &dstLocation)
	c.boxToGeneralTemp(ctx, &sourceLocation)
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opCopy, dest, dstLocation.register, sourceLocation.register)
	return varLocation{register: dest, kind: registerInt}, nil
}

// compileBuiltinNew compiles new(T) and new(expr) (Go 1.26+).
//
// Takes expression (*ast.CallExpr) which is the call expression containing the type or
// expression argument.
//
// Returns the pointer variable location and any compilation error.
func (c *compiler) compileBuiltinNew(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, fmt.Errorf("%w: new requires exactly 1 argument at %s", errCompilation, c.positionString(expression.Pos()))
	}

	tv, ok := c.info.Types[expression]
	if !ok || tv.Type == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for new at %s", errCompilation, c.positionString(expression.Pos()))
	}

	ptrType, ok := tv.Type.(*types.Pointer)
	if !ok {
		return varLocation{}, fmt.Errorf("expected *types.Pointer, got %T", tv.Type)
	}
	reflectType := c.typeToReflect(ctx, ptrType.Elem())
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}

	argTV := c.info.Types[expression.Args[0]]
	if argTV.IsType() {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opConvert, dest, 0, 1)
		c.function.emitExtension(typeIndex, 0)
		return varLocation{register: dest, kind: registerGeneral}, nil
	}

	valueLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAllocIndirect, dest, valueLocation.register, uint8(valueLocation.kind))
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinPanic compiles panic(v).
//
// Takes expression (*ast.CallExpr) which is the call expression containing the panic
// value.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinPanic(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileSingleArgGeneralTier2(ctx, expression, subOpTier2Panic, ErrCompileBuiltinPanicArgCount)
}

// compileBuiltinRecover compiles recover().
//
// Returns the recovered value location and any compilation error.
func (c *compiler) compileBuiltinRecover(_ context.Context, _ *ast.CallExpr) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Recover), dest)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinClose compiles close(ch).
//
// Takes expression (*ast.CallExpr) which is the call expression containing the channel.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinClose(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileSingleArgGeneralTier2(ctx, expression, subOpTier2ChannelClose, ErrCompileBuiltinCloseArgCount)
}

// compileSingleArgGeneralTier2 compiles a builtin that takes one argument, boxes it to
// general if needed, and emits the tier-2 drill-down form (opDrillTier1 ->
// subOpDrillTier2 -> tier2SubOp) with the boxed argument register in operand C.
//
// Takes expression (*ast.CallExpr) which is the call expression containing the argument.
// Takes tier2SubOp (subOpcodeTier2) which selects the tier-2 dispatcher arm.
// Takes errArgCount (error) which is the sentinel returned when the argument count is
// wrong.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileSingleArgGeneralTier2(ctx context.Context, expression *ast.CallExpr, tier2SubOp subOpcodeTier2, errArgCount error) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errArgCount
	}
	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &argumentLocation)
	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(tier2SubOp), argumentLocation.register)
	return varLocation{}, nil
}

// compileStructLiteral compiles a struct literal like Point{X: 1, Y: 2}.
//
// Takes lit (*ast.CompositeLit) which is the composite literal AST node.
// Takes reflectType (reflect.Type) which is the reflect type of the struct.
//
// Returns the struct variable location and any compilation error.
func (c *compiler) compileStructLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	typeIndex, err := c.function.addTypeRef(reflectType)
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)

	c.function.emit(opDrillTier1, uint8(subOpMakeMap), dest, 0)
	c.function.emitExtension(typeIndex, 0)

	for i, elt := range lit.Elts {
		if err := c.compileStructField(ctx, dest, i, elt, reflectType); err != nil {
			return varLocation{}, err
		}
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileStructField compiles a single field initialiser within a struct literal and
// emits the appropriate set-field opcode.
//
// Takes dest (uint8) which is the destination register of the struct.
// Takes positionalIndex (int) which is the positional index for unkeyed fields.
// Takes elt (ast.Expr) which is the field element expression.
// Takes reflectType (reflect.Type) which is the reflect type of the struct.
//
// Returns any compilation error.
func (c *compiler) compileStructField(ctx context.Context, dest uint8, positionalIndex int, elt ast.Expr, reflectType reflect.Type) error {
	fieldIndex, valExpr, err := resolveStructFieldIndex(positionalIndex, elt, reflectType)
	if err != nil {
		return err
	}

	valueLocation, err := c.compileExpression(ctx, valExpr)
	if err != nil {
		return err
	}
	valueLocation = c.coerceEvalBoolResult(ctx, c.info, valExpr, valueLocation)

	c.emitStructFieldSet(ctx, dest, safeconv.MustIntToUint8(fieldIndex), valueLocation, reflectType)
	return nil
}

// emitStructFieldSet emits the correct set-field opcode for the given value location,
// using the typed-bank fast path when possible.
//
// When structType is non-nil and the value's kind matches the field's declared kind, this
// routes to the tier-0 unsafe-write opcode (opSetStructFieldUint/Float/Bool) so a
// typed-bank source (uint64, float64, bool) writes directly into the field's storage at
// the known byte offset. This avoids the boxing path (emitBoxToGeneral -> opSetField ->
// handleSetField -> reflect.Value.Convert -> cvtUint) which allocs a fresh boxed Value
// per call.
//
// Takes dest (uint8) which is the destination register of the struct.
// Takes fieldIndex (uint8) which is the target field index.
// Takes valueLocation (varLocation) which is the compiled value location.
// Takes structType (reflect.Type) which is the literal's declared struct type
// (pointer-wrapped is unwrapped by the helper). Pass nil to disable the tier-0 routing.
func (c *compiler) emitStructFieldSet(ctx context.Context, dest, fieldIndex uint8, valueLocation varLocation, structType reflect.Type) {
	if valueLocation.kind == registerInt && !structFieldIsInterface(structType, fieldIndex) {
		destinationLocation := varLocation{register: dest, kind: registerGeneral}
		c.emitTyped(ctx, opSetFieldInt, destinationLocation, rawOperand(fieldIndex), valueLocation)
		return
	}
	if c.tryEmitTypedStructFieldSet(dest, fieldIndex, valueLocation, structType) {
		return
	}
	if valueLocation.kind != registerGeneral {
		generalRegister := c.scopes.alloc.allocTemp(registerGeneral)
		c.emitBoxToGeneral(ctx, generalRegister, valueLocation)
		c.function.emit(opSetField, dest, fieldIndex, generalRegister)
		c.scopes.alloc.freeTemp(registerGeneral, generalRegister)
		return
	}
	c.function.emit(opSetField, dest, fieldIndex, valueLocation.register)
}

// tryEmitTypedStructFieldSet attempts to emit one of the tier-0 or tier-1 unsafe-write
// opcodes for a typed struct field. Returns true when the typed fast path was taken; the
// caller should fall back to the boxing path on false.
//
// Takes dest (uint8) which is the destination register of the struct.
// Takes fieldIndex (uint8) which is the target field index.
// Takes valueLocation (varLocation) which is the compiled value location.
// Takes structType (reflect.Type) which is the literal's declared struct type. nil
// disables the fast path.
//
// Returns true when a fast-path opcode was emitted.
func (c *compiler) tryEmitTypedStructFieldSet(dest, fieldIndex uint8, valueLocation varLocation, structType reflect.Type) bool {
	if structType == nil || !structFieldFastPathWriteKindEnabled(valueLocation.kind) {
		return false
	}
	layoutIdx, ok := c.registerStructFieldLayoutFromReflect(structType, []int{int(fieldIndex)})
	if !ok {
		return false
	}
	layout := c.function.structLayoutTable[layoutIdx]
	if registerKind(layout.RegisterKind) != valueLocation.kind {
		return false
	}
	if structFieldLayoutIndexFitsTier0(layoutIdx) {
		if op, hasOp := pickSetStructFieldTier0Op(valueLocation.kind); hasOp {
			c.function.emit(op, dest, valueLocation.register, safeconv.MustUintToUint8(uint(layoutIdx)))
			return true
		}
	}
	if sub, hasSubOp := pickSetStructFieldUnsafeSubOp(valueLocation.kind); hasSubOp {
		c.function.emit(opDrillTier1, uint8(sub), dest, valueLocation.register)
		c.emitStructFieldLayoutExtension(layoutIdx)
		return true
	}
	return false
}

// compileTypeAssertExpression compiles a type assertion expression (x.(T)).
//
// Takes expression (*ast.TypeAssertExpr) which is the type assertion expression AST node.
//
// Returns the asserted value location and any compilation error.
func (c *compiler) compileTypeAssertExpression(ctx context.Context, expression *ast.TypeAssertExpr) (varLocation, error) {
	sourceLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	c.boxToGeneral(ctx, &sourceLocation)

	targetTypeAndValue, ok := c.info.Types[expression.Type]
	if !ok || targetTypeAndValue.Type == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for type assertion at %s", errCompilation, c.positionString(expression.Type.Pos()))
	}
	targetType := targetTypeAndValue.Type
	reflectType := c.typeAssertReflectType(ctx, targetType)
	methodNames := interfaceTargetMethodNames(targetType)
	typeIndex, err := c.function.addTypeRefWithMethods(reflectType, methodNames)
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	okRegister := c.scopes.alloc.alloc(registerInt)

	c.function.emit(opTypeAssert, dest, sourceLocation.register, okRegister)
	c.function.emitExtension(typeIndex, 1)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinPrint compiles print() or println() to opCallBuiltin.
//
// Takes expression (*ast.CallExpr) which is the call expression containing print
// arguments.
// Takes builtinID (uint8) which is the builtin identifier for the print variant.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinPrint(ctx context.Context, expression *ast.CallExpr, builtinID uint8) (varLocation, error) {
	argumentCount := len(expression.Args)
	argumentLocations := make([]varLocation, argumentCount)
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argumentLocations[i] = location
	}

	c.function.emit(opCallBuiltin, builtinID, safeconv.MustIntToUint8(argumentCount), 0)
	for _, location := range argumentLocations {
		c.function.emit(opExt, location.register, uint8(location.kind), 0)
	}

	return varLocation{}, nil
}

// compileBuiltinClear compiles clear(x) to opCallBuiltin.
//
// Takes expression (*ast.CallExpr) which is the call expression containing the argument.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinClear(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, ErrCompileBuiltinClearArgCount
	}

	argumentLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &argumentLocation)

	c.function.emit(opCallBuiltin, builtinClear, 1, 0)
	c.function.emit(opExt, argumentLocation.register, uint8(argumentLocation.kind), 0)

	return varLocation{}, nil
}

// compileBuiltinMinMax compiles min(...) or max(...) using inline comparison chains for
// int and float operands.
//
// Takes expression (*ast.CallExpr) which is the call expression containing the arguments.
// Takes isMin (bool) which controls whether this compiles min (true) or max (false).
//
// Returns the result variable location and any compilation error.
func (c *compiler) compileBuiltinMinMax(ctx context.Context, expression *ast.CallExpr, isMin bool) (varLocation, error) {
	if len(expression.Args) < 2 {
		return varLocation{}, ErrCompileBuiltinMinMaxArgCount
	}

	resultLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(resultLocation.kind)
	destLocation := varLocation{register: dest, kind: resultLocation.kind}
	c.emitMoveTyped(ctx, destLocation, resultLocation, c.staticTypeOf(expression.Args[0]))

	for _, argument := range expression.Args[1:] {
		argumentLocation, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}

		var cmpLocation varLocation
		if isMin {
			cmpLocation, err = c.emitBinaryOp(ctx, token.LSS, argumentLocation, destLocation)
		} else {
			cmpLocation, err = c.emitBinaryOp(ctx, token.GTR, argumentLocation, destLocation)
		}
		if err != nil {
			return varLocation{}, err
		}

		skipJump := c.function.emitJump(opJumpIfFalse, cmpLocation.register)
		c.emitMoveTyped(ctx, destLocation, argumentLocation, c.staticTypeOf(argument))
		c.function.patchJump(skipJump)
	}

	return destLocation, nil
}

// mapSizeHintLog2 packs a log2 map-size hint for subOpMakeMap.
//
// Encodes into the extension word's C byte: 0 means "no hint" and values
// 1..maxMapSizeHintLog2 indicate a hint of 1<<n. The hint is rounded up to the next power
// of two so the underlying map is sized to hold at least n entries without resizing.
//
// Takes n (int) which is the desired number of entries.
//
// Returns the encoded log2 hint, clamped to [0, maxMapSizeHintLog2].
func mapSizeHintLog2(n int) uint8 {
	if n <= 0 {
		return 0
	}
	lg := bits.Len(uint(n - 1))
	if lg <= 0 {
		lg = 1
	}
	if lg > maxMapSizeHintLog2 {
		lg = maxMapSizeHintLog2
	}
	return uint8(lg)
}

// typedDirectAppendSubOp returns the tier-1 sub-opcode that appends a same-bank element
// to a typed-slice-bank source slice without crossing the reflect/general bank. Returns
// false when bank is not a typed-slice bank.
//
// Takes bank (registerKind) which is the source slice's typed bank.
//
// Returns the direct sub-op and ok=true on match.
func typedDirectAppendSubOp(bank registerKind) (subOpcode, bool) {
	switch bank {
	case registerSliceInt:
		return subOpAppendSliceIntDirect, true
	case registerSliceFloat:
		return subOpAppendSliceFloatDirect, true
	case registerSliceString:
		return subOpAppendSliceStringDirect, true
	case registerSliceBool:
		return subOpAppendSliceBoolDirect, true
	case registerSliceUint:
		return subOpAppendSliceUintDirect, true
	case registerSliceByte:
		return subOpAppendSliceByteDirect, true
	default:
	}
	return 0, false
}

// typedSliceBankForElement maps a slice element type to its typed bank.
//
// Slices whose element resolves to a primitive bank route into the corresponding typed
// slice bank, eliminating reflect.Value boxing for the slice header and per-element
// access. Used by typed-aware emission paths that have proven the result stays in
// typed-bank territory; a general dataflow that mixes typed and general bank values falls
// back to the reflect path because builtin consumers (copy, append, range over interface,
// native interop) and container sinks (map values, channel elements, interface-typed
// parameters) expect reflect.Value storage.
//
// Takes sliceType (types.Type) which is the slice expression's type as resolved by
// go/types. The function inspects the underlying element type via Slice.Elem.
//
// Returns the destination bank when a typed bank applies.
// Returns true when the slice can use the typed bank, false when it must fall through to
// the reflect-based emit path.
//
//nolint:unused // gated typed-slice emission helper
func typedSliceBankForElement(sliceType types.Type) (registerKind, bool) {
	slice, ok := sliceType.Underlying().(*types.Slice)
	if !ok {
		return 0, false
	}
	if kindForType(slice.Elem()) == registerInt {
		return registerSliceInt, true
	}
	return 0, false
}

// typedSliceDirectCopySubOp maps a typed-slice register kind to its matching
// subOpCopySliceXDirect sub-op. Returns false for non- typed-slice kinds so the caller
// falls back to the reflect-based opCopy path.
//
// Takes kind (registerKind) which is the typed-slice bank under test.
//
// Returns the matching sub-op + true on a typed-slice bank, zero + false otherwise.
func typedSliceDirectCopySubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerSliceInt:
		return subOpCopySliceIntDirect, true
	case registerSliceFloat:
		return subOpCopySliceFloatDirect, true
	case registerSliceString:
		return subOpCopySliceStringDirect, true
	case registerSliceBool:
		return subOpCopySliceBoolDirect, true
	case registerSliceUint:
		return subOpCopySliceUintDirect, true
	case registerSliceByte:
		return subOpCopySliceByteDirect, true
	default:
	}
	return 0, false
}

// structFieldIsInterface reports whether a struct field has interface kind.
//
// opSetFieldInt writes an int64 directly into the field's storage slot via
// reflect.Value.SetInt, which panics with "reflect.Value.SetInt on interface Value" when
// the field is declared as an interface (any). Detecting the case at compile time routes
// such writes through the boxing path (emitBoxToGeneral + opSetField) so the value is
// wrapped in reflect.Value before assignment.
//
// Takes structType (reflect.Type) which is the struct's reflect.Type (pointer wrappers
// are peeled by the caller chain).
// Takes fieldIndex (uint8) which is the zero-based field index.
//
// Returns true when the field's reflect.Kind is Interface; false when the kind is
// concrete or the type/index lookup fails.
func structFieldIsInterface(structType reflect.Type, fieldIndex uint8) bool {
	if structType == nil {
		return false
	}
	t := structType
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	idx := int(fieldIndex)
	if idx < 0 || idx >= t.NumField() {
		return false
	}
	return t.Field(idx).Type.Kind() == reflect.Interface
}
