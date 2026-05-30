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
	"go/types"
)

// snapshotMode classifies whether a general-bank store of a value of a given static type
// must invoke valueCopyForBoundary to preserve Go's value semantics. The compiler emits
// opMoveGeneral with the matching moveGeneralMode operand (moveGeneralModeAlias,
// moveGeneralModeSnapshot, moveGeneralModeDynamic) respectively.
type snapshotMode uint8

const (
	// snapshotNever marks types whose reflect.Value header copy already matches Go's
	// reference semantics (pointer, slice, map, chan, signature, basic). The compiler emits
	// opMoveGeneral with moveGeneralModeAlias.
	snapshotNever snapshotMode = iota

	// snapshotAlways marks types that must be copied to preserve Go's value semantics
	// (struct, array). The compiler emits opMoveGeneral with moveGeneralModeSnapshot, which
	// calls the arena boundary-copy helper unconditionally, eliding the runtime kind switch.
	snapshotAlways

	// snapshotDynamic marks types whose runtime kind is not known at compile time
	// (interface, type parameter, nil). The compiler emits opMoveGeneral which performs a
	// runtime kind switch.
	snapshotDynamic
)

// compileIndexExpression compiles an index expression (a[i]).
//
// Dispatches to tryFoldStringIndex first to pre-fold s[i] when both the string subject
// and the integer index are compile-time constants. Falls through to runtime emission for
// maps via compileMapIndex, and slices/arrays/strings via compileSliceOrArrayIndex.
//
// Takes expression (*ast.IndexExpr) which is the AST index expression node.
//
// Returns varLocation holding the indexed value and any compilation error.
func (c *compiler) compileIndexExpression(ctx context.Context, expression *ast.IndexExpr) (varLocation, error) {
	if folded, ok := c.tryFoldStringIndex(ctx, expression); ok {
		return folded, nil
	}

	collectionLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	indexLocation, err := c.compileExpression(ctx, expression.Index)
	if err != nil {
		return varLocation{}, err
	}

	typeAndValue, ok := c.info.Types[expression]
	if !ok || typeAndValue.Type == nil {
		return varLocation{}, fmt.Errorf("%w: missing type information for index expression at %s", errCompilation, c.positionString(expression.Pos()))
	}
	elementKind := c.kindFor(typeAndValue.Type)
	collectionType, ok := c.underlyingTypeOf(expression.X)
	if !ok {
		return varLocation{}, fmt.Errorf("%w: missing type information for indexed collection at %s", errCompilation, c.positionString(expression.X.Pos()))
	}

	if mapType, isMap := collectionType.(*types.Map); isMap {
		return c.compileMapIndex(ctx, mapType, collectionLocation, indexLocation, elementKind)
	}
	return c.compileSliceOrArrayIndex(ctx, collectionType, collectionLocation, indexLocation, elementKind)
}

// tryFoldStringIndex pre-computes the byte at constant offset of a constant string at
// compile time. Returns the folded location and true when both subject and index are
// compile-time constants; the caller is expected to fall through to runtime emission
// otherwise.
//
// The Go expression s[i] returns a byte (uint8); the helper emits a uint-const load
// directly into the uint register bank. The peephole optimiser later rewrites this to
// subOpLoadUintConstSmall when the value fits in 8 bits.
//
// Takes ctx (context.Context) used for downstream emission.
// Takes expression (*ast.IndexExpr) which is the source-level indexing expression.
//
// Returns the folded constant location.
// Returns true when folding succeeded; false when the caller must fall through to runtime
// emission.
func (c *compiler) tryFoldStringIndex(ctx context.Context, expression *ast.IndexExpr) (varLocation, bool) {
	xTV, xOk := c.info.Types[expression.X]
	if !xOk || xTV.Value == nil || xTV.Value.Kind() != constant.String {
		return varLocation{}, false
	}
	indexTypeValue, indexOk := c.info.Types[expression.Index]
	if !indexOk || indexTypeValue.Value == nil || indexTypeValue.Value.Kind() != constant.Int {
		return varLocation{}, false
	}
	s := constant.StringVal(xTV.Value)
	index, exact := constant.Int64Val(indexTypeValue.Value)
	if !exact || index < 0 || int(index) >= len(s) {
		return varLocation{}, false
	}
	byteValue := uint64(s[index])
	register := c.scopes.alloc.alloc(registerUint)
	uintConstIndex, err := c.function.addUintConstant(byteValue)
	if err != nil {
		return varLocation{}, false
	}
	c.function.emitWide(opLoadUintConst, register, uintConstIndex)
	_ = ctx
	return varLocation{register: register, kind: registerUint}, true
}

// compileMapIndex compiles a map index expression m[k].
//
// Takes mapType (*types.Map) which is the go/types map type for the collection.
// Takes collectionLocation (varLocation) which is the varLocation of the map collection.
// Takes indexLocation (varLocation) which is the varLocation of the index key.
// Takes elementKind (registerKind) which is the expected register kind of the element.
//
// Returns varLocation holding the map element value and any compilation error.
func (c *compiler) compileMapIndex(ctx context.Context, mapType *types.Map, collectionLocation, indexLocation varLocation, elementKind registerKind) (varLocation, error) {
	keyKind := c.kindFor(mapType.Key())
	if op, ok := selectTypedMapGetOpcode(keyKind, elementKind, indexLocation.kind); ok {
		destinationRegister := c.scopes.alloc.alloc(elementKind)
		destinationLocation := varLocation{register: destinationRegister, kind: elementKind}
		c.emitTyped(ctx, op, destinationLocation, collectionLocation, indexLocation)
		return destinationLocation, nil
	}

	c.boxToGeneralTemp(ctx, &indexLocation)
	destinationRegister := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMapIndex, destinationRegister, collectionLocation.register, indexLocation.register)

	if elementKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, destinationRegister, elementKind)
	}
	return varLocation{register: destinationRegister, kind: registerGeneral}, nil
}

// compileSliceOrArrayIndex compiles a slice, array, or string index expression.
//
// Takes collectionType (types.Type) which is the go/types type of the collection.
// Takes collectionLocation (varLocation) which is the varLocation of the collection.
// Takes indexLocation (varLocation) which is the varLocation of the index.
// Takes elementKind (registerKind) which is the expected register kind of the element.
//
// Returns varLocation holding the indexed element and any compilation error.
func (c *compiler) compileSliceOrArrayIndex(ctx context.Context, collectionType types.Type, collectionLocation, indexLocation varLocation, elementKind registerKind) (varLocation, error) {
	c.ensureIntRegister(ctx, &indexLocation)
	if indexLocation.kind != registerInt {
		return varLocation{}, ErrCompileSliceIndexMustBeInteger
	}

	if basic, ok := collectionType.(*types.Basic); ok && basic.Info()&types.IsString != 0 {
		destinationRegister := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opStringIndex, destinationRegister, collectionLocation.register, indexLocation.register)
		return varLocation{register: destinationRegister, kind: registerUint}, nil
	}

	if location, ok := c.tryTypedSliceGet(ctx, collectionType, collectionLocation, indexLocation); ok {
		return location, nil
	}

	c.boxToGeneral(ctx, &collectionLocation)
	destinationRegister := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, destinationRegister, collectionLocation.register, indexLocation.register)
	if elementKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, destinationRegister, elementKind)
	}
	return varLocation{register: destinationRegister, kind: registerGeneral}, nil
}

// staticTypeOf returns an expression's static type under active substitutions.
//
// Looks up the type recorded by the type checker and applies any active type substitution
// (when the compiler is mid-specialisation), returning nil when the expression has no
// recorded type. Used by emitMoveTyped sites to pick the right opMoveGeneral snapshot
// mode and by every emit decision that consults expression types.
//
// When c.typeSubstitutions is nil (the common, non-generic case), the substitution is
// identity and the original type is returned unchanged: non-generic compilation paths see
// no behavioural change.
//
// Takes expression (ast.Expr) which is the source-level expression whose type is wanted.
// May be nil; returns nil for nil input.
//
// Returns the substituted types.Type for the expression.
// Returns nil when the expression has no recorded static type.
func (c *compiler) staticTypeOf(expression ast.Expr) types.Type {
	if c == nil || c.info == nil || expression == nil {
		return nil
	}
	tv, ok := c.info.Types[expression]
	if !ok {
		return nil
	}
	return c.substitutedType(tv.Type)
}

// tryTypedSliceGet emits a typed slice/array get if the element maps to a specialised
// register kind.
//
// Takes collectionType (types.Type) which is the go/types type of the collection.
// Takes collectionLocation (varLocation) which is the varLocation of the collection.
// Takes indexLocation (varLocation) which is the varLocation of the index.
//
// Returns varLocation and true on success, or empty varLocation and false otherwise.
func (c *compiler) tryTypedSliceGet(ctx context.Context, collectionType types.Type, collectionLocation, indexLocation varLocation) (varLocation, bool) {
	elementRegisterKind, ok := c.sliceElemRegisterKind(collectionType)
	if !ok {
		return varLocation{}, false
	}
	destinationRegister := c.scopes.alloc.alloc(elementRegisterKind)
	destinationLocation := varLocation{register: destinationRegister, kind: elementRegisterKind}
	if collectionLocation.kind == registerSliceInt && elementRegisterKind == registerInt {
		c.emitTyped(ctx, opSliceGetIntDirect, destinationLocation, collectionLocation, indexLocation)
		return destinationLocation, true
	}
	if directGetSubOp, ok := typedSliceDirectGetTier1SubOp(collectionLocation.kind); ok && elementKindForTypedSlice(collectionLocation.kind) == elementRegisterKind {
		c.function.emit(opDrillTier1, uint8(directGetSubOp), destinationLocation.register, collectionLocation.register)
		c.function.emit(opExt, indexLocation.register, 0, 0)
		return destinationLocation, true
	}
	switch elementRegisterKind {
	case registerInt:
		c.emitTyped(ctx, opSliceGetInt, destinationLocation, collectionLocation, indexLocation)
	case registerFloat:
		c.emitTyped(ctx, opSliceGetFloat, destinationLocation, collectionLocation, indexLocation)
	case registerString:
		c.emitTyped(ctx, opSliceGetString, destinationLocation, collectionLocation, indexLocation)
	case registerBool:
		c.emitTyped(ctx, opSliceGetBool, destinationLocation, collectionLocation, indexLocation)
	case registerUint:
		c.emitTyped(ctx, opSliceGetUint, destinationLocation, collectionLocation, indexLocation)
	default:
	}
	return destinationLocation, true
}

// sliceElemRegisterKind returns the register kind for a slice element.
//
// Applies to slice or array element types when they map to a specialised register.
// Specialised bodies route TypeParam elements like `V` to the concrete instantiation kind
// via c.substitutedType before classifying the element; without this routing a generic
// `[]V` body classifies its elements as registerGeneral even when the specialisation pins
// V to int/float/string/bool/uint, and the typed-slice fast-paths silently refuse,
// falling back to a general-bank indexing path that reads from the wrong bank at runtime
// (e.g. a `makeMap[K,V]` body returning zero values).
//
// Takes t (types.Type) which is the go/types type to inspect.
//
// Returns registerKind which is the specialised bank or registerGeneral.
// Returns bool which is true when a specialised bank applies.
func (c *compiler) sliceElemRegisterKind(t types.Type) (registerKind, bool) {
	t = c.substitutedType(t)
	var element types.Type
	switch u := t.Underlying().(type) {
	case *types.Slice:
		element = u.Elem()
	case *types.Array:
		element = u.Elem()
	case *types.Pointer:
		if array, ok := u.Elem().Underlying().(*types.Array); ok {
			element = array.Elem()
			break
		}
		return registerGeneral, false
	default:
		return registerGeneral, false
	}
	element = c.substitutedType(element)
	k := kindForType(element)
	if k == registerInt || k == registerFloat || k == registerString || k == registerBool || k == registerUint {
		return k, true
	}
	return registerGeneral, false
}

// selectTypedMapGetOpcode picks the typed map-get opcode for a kinds tuple.
//
// Matches (keyKind, elementKind, observedIndexKind) against the typed map-get opcode
// table; returns (0, false) when no typed opcode applies and the caller must fall back to
// the general-bank opMapIndex path. The observedIndexKind argument matches the kind of
// the compiled key expression. When it disagrees with keyKind (rare; happens with
// implicit conversions) the typed path is skipped to keep semantics simple.
//
// Takes keyKind (registerKind) which is the map's declared key kind.
// Takes elementKind (registerKind) which is the map's declared element kind.
// Takes indexKind (registerKind) which is the kind of the compiled key expression at this
// site.
//
// Returns the typed opcode and a bool indicating whether a typed opcode applies.
func selectTypedMapGetOpcode(keyKind, elementKind, indexKind registerKind) (opcode, bool) {
	if indexKind != keyKind {
		return 0, false
	}
	switch keyKind {
	case registerInt:
		switch elementKind {
		case registerInt:
			return opMapGetIntInt, true
		case registerString:
			return opMapGetIntString, true
		case registerGeneral:
			return opMapGetIntGeneral, true
		default:
		}
	case registerString:
		switch elementKind {
		case registerInt:
			return opMapGetStringInt, true
		case registerString:
			return opMapGetStringString, true
		case registerGeneral:
			return opMapGetStringGeneral, true
		default:
		}
	default:
	}
	return 0, false
}

// selectTypedMapIndexOkOpcode returns the typed map-index-with-ok opcode for `v, ok :=
// m[k]` when both key and value kinds match a supported (keyKind, valueKind) pair AND the
// compiled key expression has the same kind as the map's declared key.
//
// Takes keyKind (registerKind) which is the map's declared key kind.
// Takes valueKind (registerKind) which is the map's declared element kind.
// Takes indexKind (registerKind) which is the kind of the compiled key expression.
//
// Returns the typed opcode and a bool indicating whether a typed opcode applies.
func selectTypedMapIndexOkOpcode(keyKind, valueKind, indexKind registerKind) (opcode, bool) {
	if indexKind != keyKind {
		return 0, false
	}
	switch keyKind {
	case registerInt:
		switch valueKind {
		case registerInt:
			return opMapIndexOkIntInt, true
		case registerString:
			return opMapIndexOkIntString, true
		case registerGeneral:
			return opMapIndexOkIntGeneral, true
		default:
		}
	case registerString:
		switch valueKind {
		case registerInt:
			return opMapIndexOkStringInt, true
		case registerString:
			return opMapIndexOkStringString, true
		case registerGeneral:
			return opMapIndexOkStringGeneral, true
		default:
		}
	default:
	}
	return 0, false
}

// selectTypedMapSetOpcode returns the typed map-set opcode for the given (keyKind,
// valueKind, observedIndexKind, observedValueKind) tuple, mirroring
// selectTypedMapGetOpcode for assignments.
//
// Takes keyKind (registerKind) which is the map's declared key kind.
// Takes valueKind (registerKind) which is the map's declared element kind.
// Takes indexKind (registerKind) which is the kind of the compiled key expression.
// Takes valueObservedKind (registerKind) which is the kind of the compiled value
// expression.
//
// Returns the typed opcode and a bool indicating whether a typed opcode applies.
func selectTypedMapSetOpcode(keyKind, valueKind, indexKind, valueObservedKind registerKind) (opcode, bool) {
	if indexKind != keyKind || valueObservedKind != valueKind {
		return 0, false
	}
	switch keyKind {
	case registerInt:
		switch valueKind {
		case registerInt:
			return opMapSetIntInt, true
		case registerString:
			return opMapSetIntString, true
		default:
		}
	case registerString:
		switch valueKind {
		case registerInt:
			return opMapSetStringInt, true
		case registerString:
			return opMapSetStringString, true
		case registerGeneral:
			return opMapSetStringGeneral, true
		default:
		}
	default:
	}
	return 0, false
}

// typeIsStructOrArray reports whether t's underlying type is a struct or array, the
// value-types where copy-on-read matters per Go's value semantics. Pointers, slices,
// maps, chans, funcs are reference-typed and aliasing is the correct Go behaviour.
//
// Takes t (types.Type) which is the type to test.
//
// Returns true when t.Underlying() is *types.Struct or *types.Array.
func typeIsStructOrArray(t types.Type) bool {
	if t == nil {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Struct, *types.Array:
		return true
	default:
		return false
	}
}

// isTypeParameter reports whether t reduces to a TypeParam.
//
// A generic type parameter that the compiler must treat as registerGeneral at the
// function-body level. Used to populate CompiledFunction.parameterIsGeneric so call sites
// can detect when their callee accepts a type-erased parameter and opt into the
// generic-monomorphisation cache.
//
// Recurses through *types.Named and *types.Alias for parity with snapshotModeFor; named
// generic types have a TypeParam underlying and must classify accordingly.
//
// Takes t (types.Type) which is the parameter's declared type.
//
// Returns true when t is a type parameter at any underlying level.
func isTypeParameter(t types.Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *types.TypeParam:
		return true
	case *types.Named:
		return isTypeParameter(u.Underlying())
	default:
		return false
	}
}

// substituteType lowers t through subs, replacing TypeParams in place.
//
// Recurses through composite types (pointer, slice, array, map, chan, struct, signature,
// named with type-args). For TypeParams not present in subs, returns the original type
// unchanged. This happens for constraint-only positions in some signatures.
//
// The cache memoises walks within a single substitution invocation so deeply nested or
// recursive named types do not incur exponential cost. Pass nil to disable memoisation
// when the input is shallow.
//
// Used by generic-call body specialisation: when the compiler re-runs compileFuncBody
// with subs = {T: int}, every type-driven emit decision routes through this walker so the
// substituted body emits typed-bank opcodes (opAddInt etc.) instead of the type-erased
// general-bank path.
//
// Takes t (types.Type) which is the type to substitute. Nil returns nil.
// Takes subs (map[*types.TypeParam]types.Type) which maps each generic type parameter to
// its concrete instantiation. May be nil or empty (returns t unchanged).
// Takes cache (map[types.Type]types.Type) which memoises results. Pass nil for one-shot
// calls; pass a fresh empty map for repeated recursion.
//
// Returns the substituted types.Type, or t when the substitution would be a no-op.
func substituteType(t types.Type, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	if t == nil || len(subs) == 0 {
		return t
	}
	if cache != nil {
		if cached, ok := cache[t]; ok {
			return cached
		}
	}
	result := substituteTypeUncached(t, subs, cache)
	if cache != nil {
		cache[t] = result
	}
	return result
}

// substituteTypeUncached is the core walker. Separated from substituteType to keep the
// cache-management code out of the hot recursive switch.
//
// Takes t (types.Type) which is the type to substitute (non-nil, subs non-empty per
// substituteType's preconditions).
// Takes subs (map[*types.TypeParam]types.Type) which is the substitution map.
// Takes cache (map[types.Type]types.Type) which memoises results.
//
// Returns the substituted types.Type.
func substituteTypeUncached(t types.Type, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	switch u := t.(type) {
	case *types.TypeParam:
		if substituted, ok := subs[u]; ok {
			return substituted
		}
		return t
	case *types.Basic:
		return t
	case *types.Pointer:
		element := substituteType(u.Elem(), subs, cache)
		if element == u.Elem() {
			return t
		}
		return types.NewPointer(element)
	case *types.Slice:
		element := substituteType(u.Elem(), subs, cache)
		if element == u.Elem() {
			return t
		}
		return types.NewSlice(element)
	case *types.Array:
		return substituteArrayElement(u, subs, cache)
	case *types.Map:
		return substituteMap(u, subs, cache)
	case *types.Chan:
		return substituteChannel(u, subs, cache)
	case *types.Tuple:
		return substituteTuple(u, subs, cache)
	case *types.Signature:
		return substituteSignature(u, subs, cache)
	case *types.Struct:
		return substituteStruct(u, subs, cache)
	case *types.Named:
		return substituteNamed(u, subs, cache)
	case *types.Alias:
		return substituteType(types.Unalias(t), subs, cache)
	case *types.Interface:

		return t
	default:
		return t
	}
}

// substituteArrayElement substitutes the element type of a fixed- length array,
// preserving the array length when reconstructing.
//
// Takes u (*types.Array) which is the array type to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted array type, or u when no substitution fires.
func substituteArrayElement(u *types.Array, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	element := substituteType(u.Elem(), subs, cache)
	if element == u.Elem() {
		return u
	}
	return types.NewArray(element, u.Len())
}

// substituteMap substitutes both key and element of a map type.
//
// Takes u (*types.Map) which is the map type to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted map type, or u when no substitution fires.
func substituteMap(u *types.Map, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	key := substituteType(u.Key(), subs, cache)
	element := substituteType(u.Elem(), subs, cache)
	if key == u.Key() && element == u.Elem() {
		return u
	}
	return types.NewMap(key, element)
}

// substituteChannel substitutes the element type of a channel, preserving its direction.
//
// Takes u (*types.Chan) which is the channel type to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted channel type, or u when no substitution fires.
func substituteChannel(u *types.Chan, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	element := substituteType(u.Elem(), subs, cache)
	if element == u.Elem() {
		return u
	}
	return types.NewChan(u.Dir(), element)
}

// substituteTuple substitutes each variable in a tuple. Returns the original tuple
// unchanged when no substitution fires.
//
// Takes u (*types.Tuple) which is the tuple to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted tuple type, or u when no substitution fires.
func substituteTuple(u *types.Tuple, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) *types.Tuple {
	if u == nil || u.Len() == 0 {
		return u
	}
	changed := false
	fields := make([]*types.Var, u.Len())
	for i := range u.Len() {
		original := u.At(i)
		subType := substituteType(original.Type(), subs, cache)
		if subType == original.Type() {
			fields[i] = original
			continue
		}
		changed = true
		fields[i] = types.NewVar(original.Pos(), original.Pkg(), original.Name(), subType)
	}
	if !changed {
		return u
	}
	return types.NewTuple(fields...)
}

// substituteSignature substitutes parameter, result, and receiver types of a function
// signature.
//
// Takes u (*types.Signature) which is the signature to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted signature, or u when no substitution fires.
func substituteSignature(u *types.Signature, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	parameters := substituteTuple(u.Params(), subs, cache)
	results := substituteTuple(u.Results(), subs, cache)
	recv := u.Recv()
	if recv != nil {
		subType := substituteType(recv.Type(), subs, cache)
		if subType != recv.Type() {
			recv = types.NewVar(recv.Pos(), recv.Pkg(), recv.Name(), subType)
		}
	}
	if parameters == u.Params() && results == u.Results() && recv == u.Recv() {
		return u
	}
	return types.NewSignatureType(recv, nil, nil, parameters, results, u.Variadic())
}

// substituteStruct substitutes each field's type while preserving struct tags and field
// embedding flags.
//
// Takes u (*types.Struct) which is the struct type to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted struct type, or u when no substitution fires.
func substituteStruct(u *types.Struct, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	changed := false
	fields := make([]*types.Var, u.NumFields())
	tags := make([]string, u.NumFields())
	for i := range u.NumFields() {
		original := u.Field(i)
		subType := substituteType(original.Type(), subs, cache)
		if subType == original.Type() {
			fields[i] = original
		} else {
			changed = true
			fields[i] = types.NewField(original.Pos(), original.Pkg(), original.Name(), subType, original.Embedded())
		}
		tags[i] = u.Tag(i)
	}
	if !changed {
		return u
	}
	return types.NewStruct(fields, tags)
}

// substituteNamed substitutes the type-args of a named type and re-instantiates against
// its origin. Returns the original named type when it has no type-args or when
// instantiation fails.
//
// Takes u (*types.Named) which is the named type to substitute.
// Takes subs (map[*types.TypeParam]types.Type) which carries the active substitution map.
// Takes cache (map[types.Type]types.Type) which memoises walks across shared subtrees.
//
// Returns the substituted named type, or u when no substitution fires.
func substituteNamed(u *types.Named, subs map[*types.TypeParam]types.Type, cache map[types.Type]types.Type) types.Type {
	if u.TypeArgs() == nil || u.TypeArgs().Len() == 0 {
		return u
	}
	changed := false
	args := make([]types.Type, u.TypeArgs().Len())
	for i := range u.TypeArgs().Len() {
		original := u.TypeArgs().At(i)
		args[i] = substituteType(original, subs, cache)
		if args[i] != original {
			changed = true
		}
	}
	if !changed {
		return u
	}
	instantiated, err := types.Instantiate(nil, u.Origin(), args, false)
	if err != nil {
		return u
	}
	return instantiated
}

// snapshotModeFor picks the snapshot mode for a general-bank store of t.
//
// For pointer/slice/map/chan/func/basic kinds the reflect.Value header copy already gives
// the right semantics, so the runtime kind switch in valueCopyForBoundary is wasted work.
// For struct/array kinds the helper must run. For interface and type parameter kinds the
// runtime kind is unknown at compile time, so the conservative answer is snapshotDynamic.
//
// Recurses through *types.Named and *types.Alias so user-defined named struct types and
// Go 1.22+ type aliases are classified correctly.
//
// Takes t (types.Type) which is the source operand's static type. May be nil when the
// caller has no static type information available.
//
// Returns the snapshotMode the compiler should use to pick the move opcode.
func snapshotModeFor(t types.Type) snapshotMode {
	if t == nil {
		return snapshotDynamic
	}
	switch u := t.Underlying().(type) {
	case *types.Struct, *types.Array:
		return snapshotAlways
	case *types.Interface:
		return snapshotDynamic
	case *types.TypeParam:
		return snapshotDynamic
	case *types.Basic:
		return snapshotNever
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return snapshotNever
	case *types.Tuple:
		return snapshotDynamic
	case *types.Named:
		return snapshotModeFor(u.Underlying())
	default:
		return snapshotDynamic
	}
}
