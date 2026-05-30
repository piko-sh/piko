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
	"go/types"
	"math"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// structFieldLayoutFlagEmbedded indicates that the layout entry walks one or more
	// embedded value-struct fields before reaching the leaf. Distinguishes single-level from
	// multi-level paths without re-walking the type.
	structFieldLayoutFlagEmbedded uint8 = 1 << 0

	// structFieldLayoutFlagCycleBroken marks an any-typed leaf produced by
	// convertFieldBreakingCycles.
	//
	// The user declared the field as *Self (or a container of *Self), a self-referential
	// shape that reflect cannot construct during Self's own type build, so any is
	// substituted. The runtime held value is provably a pointer of the cycle-causing type,
	// which lets opGetStructFieldRawPointerT0 skip the abi.Type kind walk that the generic
	// Interface read otherwise has to perform. Set by buildStructFieldLayout after observing
	// the cycleBrokenTagValue on the field's reflect.StructField.Tag.
	structFieldLayoutFlagCycleBroken uint8 = 1 << 1

	// cycleBrokenTagKey is the reflect.StructField.Tag key used to mark a field whose
	// declared type was substituted to `any` by convertFieldBreakingCycles. The value is
	// irrelevant; only the key's presence matters.
	cycleBrokenTagKey = "piko_cycle_broken"

	// cycleBrokenTagValue is the canonical value attached to the cycleBrokenTagKey marker
	// tag.
	cycleBrokenTagValue = "1"

	// structFieldLayoutMaxPathDepth caps the embedded-field walk depth. 4 levels covers
	// virtually every real-world Go embedded pattern; paths deeper than this fall back to
	// the existing slow path.
	structFieldLayoutMaxPathDepth = 4
)

// structFieldLayout describes the compile-time-resolved memory layout of a single struct
// field, baked into bytecode so runtime handlers can do direct unsafe-pointer typed
// access without entering reflect.
//
// Offsets are byte counts within the deref'd struct (so a pointer receiver is
// auto-deref'd to its pointee before the offset applies). Stored as uint32 because no Go
// struct exceeds 4 GiB by language rule. The struct itself contains no unsafe-flavoured
// types so the safe build sees no taint from this declaration.
type structFieldLayout struct {
	// Offset is the byte offset of the leaf field within the deref'd struct. The unsafe
	// build adds this to the struct base pointer and casts to the field's typed pointer
	// directly.
	Offset uint32

	// TypeIndex is the typeTable index of the deref'd struct type. Kept so verifier panics
	// can name the struct involved.
	TypeIndex uint16

	// Path holds the field indices walked from the struct root to the leaf field.
	//
	// Length is PathLength; entries beyond that are zero-padded. The safe-build handler
	// walks each level via reflect.Value.Field; the unsafe-build handler ignores Path and
	// uses Offset directly. PathLength is always >= 1 for valid layouts (single-level path
	// stores the only field index in Path[0]).
	Path [structFieldLayoutMaxPathDepth]uint8

	// PathLength is the number of valid entries in Path.
	PathLength uint8

	// Kind is the reflect.Kind of the leaf field, stored as uint8 so the safe build compiles
	// without unsafe. Runtime handlers cast back to reflect.Kind for the typed load/store
	// dispatch.
	Kind uint8

	// RegisterKind is the registerKind of the leaf field. Determines which sub-op the
	// compiler emits and which register bank the runtime handler reads/writes.
	RegisterKind uint8

	// Flags carries metadata about the layout. See structFieldLayoutFlag* constants.
	Flags uint8

	// FieldTypeIndex is the typeTable index of the LEAF field's reflect.Type.
	//
	// Populated only for registerGeneral leaves (pointer / interface kinds) where the
	// handler needs the field type to construct a reflect.Value via reflect.NewAt. Zero for
	// scalar leaves whose handlers read raw memory via unsafe.Pointer + typed dereference.
	FieldTypeIndex uint16
}

// structFieldLayoutKey is the dedupe key used by the compiler to share a single
// layoutTable entry across all field-access sites targeting the same (struct type, field
// path) pair within one CompiledFunction.
type structFieldLayoutKey struct {
	// structType is the reflect.Type of the deref'd containing struct.
	structType reflect.Type

	// fieldPath is the joined reflect.StructField.Index sequence.
	fieldPath string
}

// tryResolveStructFieldLayout attempts to register a fast-path layout entry for a struct
// field access expression. Returns (layoutIndex, true) when the access is eligible for
// direct-unsafe routing and (0, false) otherwise; callers fall back to opGetField /
// opSetField when ineligible.
//
// Eligibility requires that selection.Kind() == types.FieldVal (not a method value), the
// receiver is a (possibly pointer-wrapped) struct with no interface in the chain, every
// intermediate field in selection.Index() is a value-typed struct (no mid-path pointer
// indirection), the leaf field type maps to a supported register kind (see
// structFieldLayoutSupportsKind), and a reflect.Type for the deref'd struct is obtainable
// via c.typeToReflect.
//
// Receivers containing abstract type parameters are refused: the reflect.Type derived for
// Box[T any] represents T as interface{} (16 B), but any concrete instantiation stores T
// inline at its real size, so accessing through the abstract layout would scribble
// adjacent storage. Concrete specialised bodies re-enter with a substituted receiverType
// and register their own correct layout.
//
// Takes ctx (context.Context) which is forwarded through c.typeToReflect.
// Takes selection (*types.Selection) which is the type-checker's selection metadata for
// the field access.
//
// Returns (uint16, true) with the layoutTable index on success.
// Returns (0, false) when the access is not eligible.
func (c *compiler) tryResolveStructFieldLayout(ctx context.Context, selection *types.Selection) (uint16, bool) {
	if selection == nil || selection.Kind() != types.FieldVal {
		return 0, false
	}
	fieldPath := selection.Index()
	if len(fieldPath) == 0 {
		return 0, false
	}
	receiverType := selection.Recv()
	if receiverType == nil {
		return 0, false
	}
	if _, isInterface := receiverType.Underlying().(*types.Interface); isInterface {
		return 0, false
	}
	if typesContainsTypeParam(receiverType) {
		return 0, false
	}
	reflectStructType := c.typeToReflect(ctx, receiverType)
	if reflectStructType == nil {
		return 0, false
	}
	return c.registerStructFieldLayoutFromReflect(reflectStructType, fieldPath)
}

// registerStructFieldLayoutFromReflect registers a fast-path layout entry directly from a
// reflect.Type and field path.
//
// Used by composite-literal field initialisers (compileStructField) where the struct type
// is known directly from the literal's declared type and the path is always a
// single-element positional index. Mirrors tryResolveStructFieldLayout's
// post-receiver-validation body so both call sites share the same dedupe table and the
// same per-leaf eligibility rules.
//
// Takes reflectStructType (reflect.Type) which must be a struct kind (pointer-wrapped is
// unwrapped by the caller).
// Takes fieldPath ([]int) which is the field-index chain into the struct (length 1 for
// top-level fields, >1 for embedded selections).
//
// Returns (layoutTable index, true) on success; (0, false) when the access is not
// eligible (path too deep, leaf kind unsupported, etc.).
func (c *compiler) registerStructFieldLayoutFromReflect(reflectStructType reflect.Type, fieldPath []int) (uint16, bool) {
	for reflectStructType.Kind() == reflect.Pointer {
		reflectStructType = reflectStructType.Elem()
	}
	if reflectStructType.Kind() != reflect.Struct {
		return 0, false
	}
	if len(fieldPath) > structFieldLayoutMaxPathDepth {
		return 0, false
	}

	leafType, totalOffset, encodedPath, ok := walkStructFieldPath(reflectStructType, fieldPath)
	if !ok {
		return 0, false
	}
	leafKind := leafType.Kind()
	if !structFieldLayoutSupportsKind(leafKind) {
		return 0, false
	}
	leafRegisterKind := registerKindForReflectKind(leafKind)
	if !leafRegisterKindAdmitted(leafRegisterKind, leafKind) {
		return 0, false
	}

	layout, ok := c.buildStructFieldLayout(reflectStructType, leafType, leafRegisterKind, leafKind, totalOffset, encodedPath, fieldPath)
	if !ok {
		return 0, false
	}
	return c.internStructFieldLayout(reflectStructType, fieldPath, layout)
}

// buildStructFieldLayout constructs the structFieldLayout record from the walk results
// and the leaf metadata.
//
// Takes reflectStructType (reflect.Type) which is the root struct reflect.Type the access
// is rooted at.
// Takes leafType (reflect.Type) which is the reflect.Type of the resolved leaf field.
// Takes leafRegisterKind (registerKind) which is the register kind of the leaf field.
// Takes leafKind (reflect.Kind) which is the reflect kind of the leaf field.
// Takes totalOffset (uint32) which is the cumulative byte offset of the leaf within the
// deref'd struct.
// Takes encodedPath which is the field-index path padded to
// structFieldLayoutMaxPathDepth.
// Takes fieldPath ([]int) which is the original field-index chain from the type-checker.
//
// Returns the populated structFieldLayout record.
func (c *compiler) buildStructFieldLayout(
	reflectStructType, leafType reflect.Type,
	leafRegisterKind registerKind,
	leafKind reflect.Kind,
	totalOffset uint32,
	encodedPath [structFieldLayoutMaxPathDepth]uint8,
	fieldPath []int,
) (structFieldLayout, bool) {
	typeIndex, err := c.function.addTypeRef(reflectStructType)
	if err != nil {
		c.recordStickyError(err)
		return structFieldLayout{}, false
	}
	var flags uint8
	if len(fieldPath) > 1 {
		flags |= structFieldLayoutFlagEmbedded
	}
	if leafFieldIsCycleBroken(reflectStructType, fieldPath) {
		flags |= structFieldLayoutFlagCycleBroken
	}
	var fieldTypeIndex uint16
	if leafRegisterKind == registerGeneral {
		index, leafErr := c.function.addTypeRef(leafType)
		if leafErr != nil {
			c.recordStickyError(leafErr)
			return structFieldLayout{}, false
		}
		fieldTypeIndex = index
	}
	return structFieldLayout{
		Offset:         totalOffset,
		TypeIndex:      typeIndex,
		Path:           encodedPath,
		PathLength:     safeconv.MustIntToUint8(len(fieldPath)),
		Kind:           safeconv.UintToUint8(uint(leafKind)),
		RegisterKind:   uint8(leafRegisterKind),
		Flags:          flags,
		FieldTypeIndex: fieldTypeIndex,
	}, true
}

// internStructFieldLayout dedupes layout against c.structLayoutIndex.
//
// When the same (struct type, field path) pair was already registered, the existing index
// is returned; otherwise a fresh layout is appended to the structLayoutTable and its
// newly allocated index is returned.
//
// Takes reflectStructType (reflect.Type) which is the root struct reflect.Type the layout
// is keyed on.
// Takes fieldPath ([]int) which is the field-index chain forming the second half of the
// key.
// Takes layout (structFieldLayout) which is the layout record to intern when no match
// exists.
//
// Returns the structLayoutTable index (existing or newly allocated) and true on success.
func (c *compiler) internStructFieldLayout(reflectStructType reflect.Type, fieldPath []int, layout structFieldLayout) (uint16, bool) {
	key := structFieldLayoutKey{
		structType: reflectStructType,
		fieldPath:  encodeFieldPath(fieldPath),
	}
	if c.structLayoutIndex == nil {
		c.structLayoutIndex = make(map[structFieldLayoutKey]uint16)
	}
	if existingIndex, ok := c.structLayoutIndex[key]; ok {
		return existingIndex, true
	}
	index := safeconv.MustIntToUint16(len(c.function.structLayoutTable))
	c.function.structLayoutTable = append(c.function.structLayoutTable, layout)
	c.structLayoutIndex[key] = index
	return index, true
}

// structFieldSliceSubOpPair groups the typed-slice struct-field tier-1 sub-ops for a
// single element-bank: the GET (read) sub-op, the SET (write) sub-op, and the typed-slice
// register bank the operand reads/writes. Indexed by element basic-kind in
// structFieldSliceSubOpByElemKind.
type structFieldSliceSubOpPair struct {
	// get is the read-side tier-1 sub-op for the slice element bank.
	get subOpcode

	// set is the write-side tier-1 sub-op for the slice element bank.
	set subOpcode

	// bank is the typed-slice register bank read or written by the sub-ops.
	bank registerKind
}

// emitStructFieldLayoutExtension emits the opExt extension word carrying the 16-bit
// structLayoutTable index following a subOpGet/SetStructFieldXxx primary word.
//
// Takes layoutIndex (uint16) which is the layoutTable index to encode.
func (c *compiler) emitStructFieldLayoutExtension(layoutIndex uint16) {
	c.function.emit(opExt, uint8(layoutIndex&0xFF), uint8(layoutIndex>>8), 0)
}

// walkStructFieldPath walks the field-index chain into the struct.
//
// Takes reflectStructType (reflect.Type) which is the root struct type to descend from.
// Takes fieldPath ([]int) which is the field-index chain into the struct.
//
// Returns the leaf reflect.Type reached at the end of the chain, the cumulative byte
// offset of the leaf within the deref'd struct, the encoded path padded to
// structFieldLayoutMaxPathDepth, and true when every step lands on a struct field within
// bounds and the offset fits in uint32; false otherwise.
func walkStructFieldPath(reflectStructType reflect.Type, fieldPath []int) (reflect.Type, uint32, [structFieldLayoutMaxPathDepth]uint8, bool) {
	var totalOffset uint32
	var encodedPath [structFieldLayoutMaxPathDepth]uint8
	currentType := reflectStructType
	for depth, fieldIndex := range fieldPath {
		if currentType.Kind() != reflect.Struct {
			return nil, 0, encodedPath, false
		}
		if fieldIndex < 0 || fieldIndex >= currentType.NumField() {
			return nil, 0, encodedPath, false
		}
		field := currentType.Field(fieldIndex)
		if field.Offset > math.MaxUint32 {
			return nil, 0, encodedPath, false
		}
		totalOffset += uint32(field.Offset)
		encodedPath[depth] = safeconv.MustIntToUint8(fieldIndex)
		if depth == len(fieldPath)-1 {
			currentType = field.Type
			continue
		}
		next := field.Type
		if next.Kind() == reflect.Pointer || next.Kind() != reflect.Struct {
			return nil, 0, encodedPath, false
		}
		currentType = next
	}
	return currentType, totalOffset, encodedPath, true
}

// leafRegisterKindAdmitted reports whether a leaf is eligible for the fast path.
//
// General-bank leaves are admitted only when the leaf type uses a representation the
// unsafe handler in tryGetStructFieldUnsafe knows how to snapshot: Pointer (snapshot via
// snapshotPointerLeaf); Interface (snapshot via tryGetStructInterfaceField); Slice
// (snapshot via vm.acquireSliceSnapshot); or Map / Chan / Func (snapshot via
// unsafeDirectIfaceKindValue, the same one-word read the generic handleGetField path
// performs).
//
// Takes leafRegisterKind (registerKind) which is the register kind of the leaf field.
// Takes leafKind (reflect.Kind) which is the reflect kind of the leaf field.
//
// Returns true when the leaf is admitted by the fast-path eligibility rules; false
// otherwise.
func leafRegisterKindAdmitted(leafRegisterKind registerKind, leafKind reflect.Kind) bool {
	if leafRegisterKind != registerGeneral {
		return true
	}
	switch leafKind {
	case reflect.Pointer,
		reflect.Interface,
		reflect.Slice,
		reflect.Map,
		reflect.Chan,
		reflect.Func:
		return true
	default:
	}
	return false
}

// leafFieldIsCycleBroken reports whether the leaf field carries the cycle-broken marker
// tag.
//
// The marker is stamped by buildStructFields when the declared field type referenced the
// still-under-construction type. Layouts flagged this way are eligible for the
// opGetStructFieldRawPointerT0 fast path because the runtime held value is provably a
// pointer.
//
// Takes reflectStructType (reflect.Type) which is the root struct reflect.Type to walk
// from.
// Takes fieldPath ([]int) which is the field-index chain leading to the leaf.
//
// Returns true when the leaf carries the cycle-broken marker tag; false otherwise.
func leafFieldIsCycleBroken(reflectStructType reflect.Type, fieldPath []int) bool {
	if len(fieldPath) == 0 {
		return false
	}
	currentType := reflectStructType
	for currentType.Kind() == reflect.Pointer {
		currentType = currentType.Elem()
	}
	for step, fieldIndex := range fieldPath {
		if currentType.Kind() != reflect.Struct {
			return false
		}
		if fieldIndex >= currentType.NumField() {
			return false
		}
		field := currentType.Field(fieldIndex)
		if step == len(fieldPath)-1 {
			return field.Tag.Get(cycleBrokenTagKey) == cycleBrokenTagValue
		}
		currentType = field.Type
		for currentType.Kind() == reflect.Pointer {
			currentType = currentType.Elem()
		}
	}
	return false
}

// typesContainsTypeParam reports whether t contains a type parameter.
//
// Walking the type tree picks up generic-body cases that isTypeParameter doesn't recurse
// into - notably *Named with concrete-named outer but abstract type args, struct fields,
// pointer/slice/map/chan element types. Used by tryResolveStructFieldLayout to refuse
// registering a layout whose reflect.Type representation is type-erased; the
// concrete-specialised body re-enters with a substituted receiverType and registers its
// own correct layout.
//
// Takes t (types.Type) which is the type to scan.
//
// Returns true when t or any nested composite is a *types.TypeParam.
func typesContainsTypeParam(t types.Type) bool {
	const initialSeenHint = 8
	return typesContainsTypeParamSeen(t, make(map[types.Type]struct{}, initialSeenHint))
}

// typesContainsTypeParamSeen is the cycle-guarded body of typesContainsTypeParam.
//
// Takes t (types.Type) which is the type to scan.
// Takes seen (map[types.Type]struct{}) which records visited nodes to break
// self-referential type graphs.
//
// Returns true when t or any nested composite is a *types.TypeParam.
func typesContainsTypeParamSeen(t types.Type, seen map[types.Type]struct{}) bool {
	if t == nil {
		return false
	}
	if _, ok := seen[t]; ok {
		return false
	}
	seen[t] = struct{}{}
	switch u := t.(type) {
	case *types.TypeParam:
		return true
	case *types.Named:
		return namedContainsTypeParam(u, seen)
	case *types.Pointer:
		return typesContainsTypeParamSeen(u.Elem(), seen)
	case *types.Slice:
		return typesContainsTypeParamSeen(u.Elem(), seen)
	case *types.Array:
		return typesContainsTypeParamSeen(u.Elem(), seen)
	case *types.Map:
		return typesContainsTypeParamSeen(u.Key(), seen) || typesContainsTypeParamSeen(u.Elem(), seen)
	case *types.Chan:
		return typesContainsTypeParamSeen(u.Elem(), seen)
	case *types.Struct:
		return structContainsTypeParam(u, seen)
	case *types.Alias:
		return typesContainsTypeParamSeen(u.Underlying(), seen)
	}
	return false
}

// namedContainsTypeParam scans the named type's type arguments and underlying definition
// for any type parameter.
//
// Takes u (*types.Named) which is the named type to scan.
// Takes seen (map[types.Type]struct{}) which is the visited-node set used to break
// self-referential type graphs.
//
// Returns true when u's arguments or underlying definition contain a type parameter;
// false otherwise.
func namedContainsTypeParam(u *types.Named, seen map[types.Type]struct{}) bool {
	if arguments := u.TypeArgs(); arguments != nil {
		for t := range arguments.Types() {
			if typesContainsTypeParamSeen(t, seen) {
				return true
			}
		}
	}
	return typesContainsTypeParamSeen(u.Underlying(), seen)
}

// structContainsTypeParam scans every field type of u for a type parameter.
//
// Takes u (*types.Struct) which is the struct type to scan.
// Takes seen (map[types.Type]struct{}) which is the visited-node set used to break
// self-referential type graphs.
//
// Returns true when any field type contains a type parameter; false otherwise.
func structContainsTypeParam(u *types.Struct, seen map[types.Type]struct{}) bool {
	for field := range u.Fields() {
		if typesContainsTypeParamSeen(field.Type(), seen) {
			return true
		}
	}
	return false
}

// structFieldLayoutSupportsKind reports whether a reflect.Kind is one of the leaf kinds
// the fast path supports.
//
// Scalar kinds (Int*, Uint*, Float*, Bool, String) route through the typed sub-ops.
// Pointer, Interface, Slice, Map, Chan, and Func leaves route to opGetStructFieldGeneral,
// which snapshots the underlying value (via reflect.NewAt for pointer/interface, header
// copy for slice, direct-iface-kind read for map/chan/func) so the returned reflect.Value
// detaches from the struct's storage.
//
// Takes kind (reflect.Kind) which is the leaf field's kind.
//
// Returns true when the kind is in the supported set.
func structFieldLayoutSupportsKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Bool,
		reflect.String,
		reflect.Pointer,
		reflect.Interface,
		reflect.Slice,
		reflect.Map, reflect.Chan, reflect.Func:
		return true
	default:
	}
	return false
}

// registerKindForReflectKind maps a reflect.Kind to the matching registerKind for the
// fast-path sub-op selection.
//
// Takes kind (reflect.Kind) which is the leaf field's kind.
//
// Returns the matching registerKind, or registerGeneral when no typed bank applies.
func registerKindForReflectKind(kind reflect.Kind) registerKind {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return registerInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return registerUint
	case reflect.Float32, reflect.Float64:
		return registerFloat
	case reflect.Bool:
		return registerBool
	case reflect.String:
		return registerString
	default:
	}
	return registerGeneral
}

// pickGetStructFieldUnsafeSubOp returns the read-side sub-opcode for the given leaf
// register kind.
//
// Takes kind (registerKind) which is the leaf field's register kind.
//
// Returns the matching subOpGetStructFieldXxx sub-op and true on success; (0, false) when
// no sub-op exists for the kind.
func pickGetStructFieldUnsafeSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerInt:
		return subOpGetStructFieldInt, true
	case registerUint:
		return subOpGetStructFieldUint, true
	case registerFloat:
		return subOpGetStructFieldFloat, true
	case registerBool:
		return subOpGetStructFieldBool, true
	case registerString:
		return subOpGetStructFieldString, true
	default:
	}
	return 0, false
}

// pickSetStructFieldUnsafeSubOp returns the write-side sub-opcode for the given leaf
// register kind.
//
// Takes kind (registerKind) which is the leaf field's register kind.
//
// Returns the matching subOpSetStructFieldXxx sub-op and true on success; (0, false) when
// no sub-op exists for the kind.
func pickSetStructFieldUnsafeSubOp(kind registerKind) (subOpcode, bool) {
	switch kind {
	case registerInt:
		return subOpSetStructFieldInt, true
	case registerUint:
		return subOpSetStructFieldUint, true
	case registerFloat:
		return subOpSetStructFieldFloat, true
	case registerBool:
		return subOpSetStructFieldBool, true
	case registerString:
		return subOpSetStructFieldString, true
	default:
	}
	return 0, false
}

// lookupStructFieldSliceSubOpPair returns the typed-slice sub-op pair.
//
// Matches a slice-typed struct field whose element kind aligns with a canonical
// typed-slice bank width. Returns (pair, true) on a match; (zero, false) for non-slice
// fields, non-basic element types, or narrower-than-canonical widths.
//
// Only canonical 64-bit element widths (int64 / uint64 / float64) plus the
// degenerate-width string / bool / byte cases are admitted. Narrower-width slice elements
// (int8 / int16 / int32, uint16 / uint32) intentionally fall through to the general-bank
// reflect path; unboxToTypedIntSlice widens at any later cross-bank boundary.
//
// Takes fieldType (types.Type) which is the struct field's static type.
//
// Returns structFieldSliceSubOpPair which carries get/set/bank on a match.
// Returns bool which is true when a typed-slice match exists.
func lookupStructFieldSliceSubOpPair(fieldType types.Type) (structFieldSliceSubOpPair, bool) {
	slice, ok := fieldType.Underlying().(*types.Slice)
	if !ok {
		return structFieldSliceSubOpPair{}, false
	}
	basic, ok := slice.Elem().Underlying().(*types.Basic)
	if !ok {
		return structFieldSliceSubOpPair{}, false
	}

	switch basic.Kind() { //nolint:exhaustive // intentional partial: only canonical-width typed-slice element kinds admit a direct alias
	case types.Int, types.Int64:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceInt, set: subOpSetStructFieldSliceInt, bank: registerSliceInt}, true
	case types.Float64:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceFloat, set: subOpSetStructFieldSliceFloat, bank: registerSliceFloat}, true
	case types.Uint, types.Uint64:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceUint, set: subOpSetStructFieldSliceUint, bank: registerSliceUint}, true
	case types.String:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceString, set: subOpSetStructFieldSliceString, bank: registerSliceString}, true
	case types.Bool:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceBool, set: subOpSetStructFieldSliceBool, bank: registerSliceBool}, true
	case types.Uint8:
		return structFieldSliceSubOpPair{get: subOpGetStructFieldSliceByte, set: subOpSetStructFieldSliceByte, bank: registerSliceByte}, true
	}
	return structFieldSliceSubOpPair{}, false
}

// pickGetStructFieldSliceSubOp returns the read-side typed-slice sub-op.
//
// Matches slice-typed struct fields whose element kind aligns with one of the typed-slice
// banks' canonical storage widths. Returns (sub-opcode, destination-bank-kind, true) on a
// match. Narrower-width slice element types (int8/int16/int32 / uint16/uint32 / etc. when
// the bank's canonical storage is 64-bit) fall through to (0, registerGeneral, false):
// the caller emits the general-bank snapshot reader, and unboxToTypedIntSlice handles
// widening at any later cross-bank boundary.
//
// Takes fieldType (types.Type) which is the field's static slice type.
//
// Returns the sub-opcode, the destination bank's registerKind, and true on a typed-slice
// match; (0, registerGeneral, false) otherwise.
func pickGetStructFieldSliceSubOp(fieldType types.Type) (subOpcode, registerKind, bool) {
	pair, ok := lookupStructFieldSliceSubOpPair(fieldType)
	if !ok {
		return 0, registerGeneral, false
	}
	return pair.get, pair.bank, true
}

// pickSetStructFieldSliceSubOp returns the write-side tier-1 sub-op for a slice-typed
// struct field. Same canonical-width gate as pickGetStructFieldSliceSubOp.
//
// Takes fieldType (types.Type) which is the field's static slice type.
//
// Returns the sub-opcode, the source bank's registerKind, and true on a typed-slice
// match; (0, registerGeneral, false) otherwise.
func pickSetStructFieldSliceSubOp(fieldType types.Type) (subOpcode, registerKind, bool) {
	pair, ok := lookupStructFieldSliceSubOpPair(fieldType)
	if !ok {
		return 0, registerGeneral, false
	}
	return pair.set, pair.bank, true
}

// pickGetStructFieldTier0Op returns the read-side tier-0 opcode for the given leaf
// register kind. The tier-0 form is a single bytecode word (no opExt extension) and is
// preferred when the layoutTable index fits in a uint8.
//
// String reads have no tier-0 variant: the string-write path's reflect.NewAt+SetString
// cost dominates so the dispatch saving is less impactful, and keeping String at tier-1
// saves opcode space.
//
// Takes kind (registerKind) which is the leaf field's register kind.
//
// Returns the matching opcode and true on success; (0, false) when no tier-0 form exists
// for the kind.
func pickGetStructFieldTier0Op(kind registerKind) (opcode, bool) {
	switch kind {
	case registerInt:
		return opGetStructFieldIntT0, true
	case registerUint:
		return opGetStructFieldUint, true
	case registerFloat:
		return opGetStructFieldFloat, true
	case registerBool:
		return opGetStructFieldBool, true
	case registerGeneral:
		return opGetStructFieldGeneral, true
	default:
	}
	return 0, false
}

// pickSetStructFieldTier0Op returns the write-side tier-0 opcode for the given leaf
// register kind. String writes have no tier-0 variant (see pickGetStructFieldTier0Op).
//
// Takes kind (registerKind) which is the leaf field's register kind.
//
// Returns the matching opcode and true on success; (0, false) when no tier-0 form exists
// for the kind.
func pickSetStructFieldTier0Op(kind registerKind) (opcode, bool) {
	switch kind {
	case registerInt:
		return opSetStructFieldIntT0, true
	case registerUint:
		return opSetStructFieldUint, true
	case registerFloat:
		return opSetStructFieldFloat, true
	case registerBool:
		return opSetStructFieldBool, true
	case registerGeneral:
		return opSetStructFieldGeneral, true
	default:
	}
	return 0, false
}

// structFieldLayoutIndexFitsTier0 reports whether the index fits in a tier-0 operand
// byte.
//
// Tier-0 struct-field opcodes encode the layout index in a single uint8 operand; when the
// function's layoutTable has more than 256 entries and the index is >= 256, the compiler
// must fall back to the tier-1 sub-op form (which encodes the index in the following
// opExt extension word).
//
// Takes layoutIndex (uint16) which is the layoutTable index.
//
// Returns true when the index fits in a uint8 operand.
func structFieldLayoutIndexFitsTier0(layoutIndex uint16) bool {
	const tier0MaxLayoutIndex uint16 = 256
	return layoutIndex < tier0MaxLayoutIndex
}

// encodeFieldPath serialises an []int field path into a compact string suitable for use
// as a dedupe map key. Used inside structFieldLayoutKey to compare embedded paths without
// allocating a per-key slice.
//
// Takes path ([]int) which is the selection.Index() field path.
//
// Returns a stable string encoding suitable for map-key comparison.
func encodeFieldPath(path []int) string {
	const maxDecimalDigitsPerInt = 10
	const reservedBytesPerEntry = 5
	const decimalBase = 10
	buffer := make([]byte, 0, len(path)*reservedBytesPerEntry)
	for _, index := range path {
		v := index
		if v < 0 {
			buffer = append(buffer, '-')
			v = -v
		}
		var digits [maxDecimalDigitsPerInt]byte
		n := 0
		if v == 0 {
			digits[0] = '0'
			n = 1
		} else {
			for v > 0 {
				digits[n] = byte('0' + v%decimalBase)
				v /= decimalBase
				n++
			}
		}
		for i := n - 1; i >= 0; i-- {
			buffer = append(buffer, digits[i])
		}
		buffer = append(buffer, ',')
	}
	return string(buffer)
}
