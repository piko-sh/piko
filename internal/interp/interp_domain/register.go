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
	"go/types"
	"reflect"
)

// registerKind identifies which typed register bank a value belongs to. Using separate
// banks for common primitive types avoids the overhead of boxing/unboxing values in
// reflect.Value for the majority of operations.
type registerKind uint8

const (
	// registerInt stores int64 values. All Go signed integer types (int, int8, int16, int32,
	// int64) and untyped int/rune are stored here, using int64 as the common representation.
	registerInt registerKind = iota

	// registerFloat stores float64 values. Both float32 and float64 are stored here, with
	// float32 promoted to float64.
	registerFloat

	// registerString stores string values natively.
	registerString

	// registerGeneral stores reflect.Value for all other types: interfaces, pointers,
	// slices, maps, arrays, structs, channels, and functions.
	registerGeneral

	// registerBool stores bool values natively.
	registerBool

	// registerUint stores uint64 values. All Go unsigned integer types (uint, uint8, uint16,
	// uint32, uint64, uintptr) are stored here.
	registerUint

	// registerComplex stores complex128 values. Both complex64 and complex128 are stored
	// here, with complex64 promoted to complex128.
	registerComplex

	// registerSliceInt stores []int64 slice headers natively. The compiler selects this bank
	// when the slice element kind resolves to registerInt at compile time, eliminating
	// reflect.Value boxing for the slice header and per-element access.
	registerSliceInt

	// registerSliceFloat stores []float64 slice headers natively for slices whose element
	// kind resolves to registerFloat.
	registerSliceFloat

	// registerSliceString stores []string slice headers natively for slices whose element
	// kind resolves to registerString.
	registerSliceString

	// registerSliceBool stores []bool slice headers natively for slices whose element kind
	// resolves to registerBool.
	registerSliceBool

	// registerSliceUint stores []uint64 slice headers natively for slices whose element kind
	// resolves to registerUint.
	registerSliceUint

	// registerSliceByte stores []byte slice headers natively.
	//
	// Distinct from registerSliceUint because the element width is 1 byte instead of 8, and
	// byte iteration is the inner-loop hot path for parsers, JSON, and brainfuck. Keeping
	// the storage tight lets the ASM tier-1 handlers read elements with a single MOVBLZX
	// instead of going through reflect.Value.Index.
	registerSliceByte
)

const (
	// NumRegisterKinds is the number of register bank types.
	NumRegisterKinds = 13

	// narrowIntegerBitWidth8 selects an 8-bit truncation for int8 and uint8 arithmetic
	// results emitted via opTruncateNarrow.
	narrowIntegerBitWidth8 uint8 = 8

	// narrowIntegerBitWidth16 selects a 16-bit truncation for int16 and uint16 arithmetic
	// results emitted via opTruncateNarrow.
	narrowIntegerBitWidth16 uint8 = 16

	// narrowIntegerBitWidth32 selects a 32-bit truncation for int32 and uint32 arithmetic
	// results emitted via opTruncateNarrow. Width 64 is intentionally absent because
	// int64/uint64 already hold the full declared width.
	narrowIntegerBitWidth32 uint8 = 32
)

// String returns the human-readable name of the register kind.
//
// Returns the register kind name as a string.
func (k registerKind) String() string {
	switch k {
	case registerInt:
		return "int"
	case registerFloat:
		return "float"
	case registerString:
		return "string"
	case registerGeneral:
		return "general"
	case registerBool:
		return "bool"
	case registerUint:
		return "uint"
	case registerComplex:
		return "complex"
	case registerSliceInt:
		return "sliceInt"
	case registerSliceFloat:
		return "sliceFloat"
	case registerSliceString:
		return "sliceString"
	case registerSliceBool:
		return "sliceBool"
	case registerSliceUint:
		return "sliceUint"
	case registerSliceByte:
		return "sliceByte"
	default:
		return "unknown"
	}
}

// Registers holds the typed register banks for a call frame.
//
// Bank ordering must match the registerKind iota values (registerInt=0, registerFloat=1,
// ..., registerSliceInt=7). The ASM dispatch tier reads the int/float/string/bool/uint
// banks at fixed byte offsets pinned by asm_dispatch_offsets.h and verified by
// TestCallFrameOffsets; new banks must be appended at the end so existing offsets remain
// stable. The general and complex banks are not ASM-accessed.
type Registers struct {
	// ints stores int64 values for the integer register bank.
	ints []int64

	// floats stores float64 values for the float register bank.
	floats []float64

	// strings stores string values for the string register bank.
	strings []string

	// general stores reflect.Value values for the general register bank.
	general []reflect.Value

	// bools stores bool values for the boolean register bank.
	bools []bool

	// uints stores uint64 values for the unsigned integer register bank.
	uints []uint64

	// complex stores complex128 values for the complex register bank.
	complex []complex128

	// slicesInt stores []int64 slice headers for the typed-slice-int register bank. The
	// compiler routes make([]T, ...) where T resolves to int-kind into this bank instead of
	// boxing a reflect.Value into the general bank, removing per-element reflect.Value
	// allocations on get/set.
	slicesInt [][]int64

	// slicesFloat stores []float64 slice headers for the typed slice float bank.
	slicesFloat [][]float64

	// slicesString stores []string slice headers for the typed slice string bank.
	slicesString [][]string

	// slicesBool stores []bool slice headers for the typed slice bool bank.
	slicesBool [][]bool

	// slicesUint stores []uint64 slice headers for the typed slice uint bank.
	slicesUint [][]uint64

	// slicesByte stores []byte slice headers for the typed slice byte bank. Backing storage
	// comes from RegisterArena.byteSlab via AllocByteBacking - the same arena slab that
	// backs make([]byte, n) in the general path - so make/append/slice ops on byte slices
	// avoid heap allocation entirely on the fast path.
	slicesByte [][]byte
}

// NewRegistersForBench is an exported wrapper for benchmarking direct allocation vs arena
// allocation.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of registers per bank.
//
// Returns a freshly allocated register file.
func NewRegistersForBench(numRegs [NumRegisterKinds]uint32) Registers {
	return newRegisters(numRegs)
}

// newRegisters creates a register file sized for a compiled function.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of registers per bank.
//
// Returns a freshly allocated register file.
func newRegisters(numRegs [NumRegisterKinds]uint32) Registers {
	return Registers{
		ints:         make([]int64, numRegs[registerInt]),
		floats:       make([]float64, numRegs[registerFloat]),
		strings:      make([]string, numRegs[registerString]),
		general:      make([]reflect.Value, numRegs[registerGeneral]),
		bools:        make([]bool, numRegs[registerBool]),
		uints:        make([]uint64, numRegs[registerUint]),
		complex:      make([]complex128, numRegs[registerComplex]),
		slicesInt:    make([][]int64, numRegs[registerSliceInt]),
		slicesFloat:  make([][]float64, numRegs[registerSliceFloat]),
		slicesString: make([][]string, numRegs[registerSliceString]),
		slicesBool:   make([][]bool, numRegs[registerSliceBool]),
		slicesUint:   make([][]uint64, numRegs[registerSliceUint]),
		slicesByte:   make([][]byte, numRegs[registerSliceByte]),
	}
}

// kindForType determines the register kind for a given Go type at value-context sites
// (assignment RHS, expression results, etc.).
//
// Takes t (types.Type) which is the Go type to classify.
//
// Returns the register kind for the type's register bank.
//
// Slices are classified as registerGeneral here. The typed-storage banks
// (registerSliceInt et al.) hold slice headers only for values whose origin and lifetime
// the compiler has validated. Three compile-site entry points expose the typed-bank
// classification: kindForTypedSlice for typed allocation sites (make([]T, ...)) where the
// storage type matches the typed bank by construction; kindForCallSlot for function
// parameter and return slots whose element type's STORAGE exactly matches the typed bank
// (strictly []int64, []float64, []string, []bool, []uint64, []byte; no narrow ints, no
// named slice types) plumbed through argCopyProgram's typed copy ops, with
// storage-incompatible slice types staying on the general bank to preserve mutation
// aliasing across the call boundary; and the escape analysis at classifyTypedSliceLocals,
// which promotes typed-bank eligible locals from the general bank based on body-wide
// usage validation.
func kindForType(t types.Type) registerKind {
	t = t.Underlying()

	if basic, ok := t.(*types.Basic); ok {
		return kindForBasic(basic.Kind())
	}

	return registerGeneral
}

// kindForCallSlot returns the register kind for a call-slot type.
//
// Slice element types of fixed width share a typed-slice bank with their wider 64-bit
// sibling: []int16 and []int32 join []int64 on registerSliceInt; []uint16 and []uint32
// join []uint64 on registerSliceUint. []float64 / []string / []bool have one storage
// width each; []uint8 routes to its compact-stride registerSliceByte bank.
//
// Sub-int widths are safe to share with their 64-bit sibling because the typed-bank ABI
// stores values as int64 or uint64 regardless of the declared element width:
// asmCallMakeSliceInt and the typed-direct readers and writers handle widening on read
// (via the opTruncateNarrow pass) and the call-boundary copy helpers (boxScalarToGeneral
// / unboxGeneralToScalar) consult the user's element kind through the call-site metadata
// to preserve reflect.Type identity when crossing into the general bank.
//
// Platform-int []int / []uint: piko targets amd64, arm64, and wasm exclusively (all
// 64-bit ABIs where int and uint are 64-bit). asmCallMakeSliceInt
// (asm_call_trampolines.go:378-385) already produces []int64-backed values for
// `make([]int, n)` locals, and opPackTyped preserves the declared reflect.Type identity
// across cross-bank moves, so admitting types.Int at this site lets `[]int` reach the
// typed banks safely.
//
// Note this is stricter than kindForTypedSlice, which is permissive because typed-bank
// LOCALS (from make([]int32, ...)) never cross a boundary in the first place - escape
// analysis demotes them. Parameter/return slots DO cross boundaries, so the storage-type
// invariant must hold. The body-usage gate at parameterRuntimeKind (see
// classifyTypedSliceParamNames) still demotes any typed-slice parameter whose body uses
// it in patterns the typed bank cannot represent (append, copy, &xs, container store,
// etc.); the gate is width-agnostic and applies uniformly to all six typed-slice banks.
//
// Takes t (types.Type) which is the Go type of the call slot.
//
// Returns the register kind for the parameter or return slot.
func kindForCallSlot(t types.Type) registerKind {
	slice, ok := t.(*types.Slice)
	if !ok {
		return kindForType(t)
	}
	elementBasic, basicOk := slice.Elem().(*types.Basic)
	if !basicOk {
		return registerGeneral
	}
	switch elementBasic.Kind() {
	case types.Int, types.Int64:

		return registerSliceInt
	case types.Float64:
		return registerSliceFloat
	case types.String:
		return registerSliceString
	case types.Bool:
		return registerSliceBool
	case types.Uint, types.Uint64:

		return registerSliceUint
	case types.Uint8:
		return registerSliceByte
	default:
	}

	return registerGeneral
}

// kindPromotionContext carries inputs for the typed-bank promotion gate.
//
// kindForPromotedSlot reads it to make a typed-bank kind decision consistent across the
// five boundary sites: kindForCallSlot (call/return), tryEmitSelectorFieldSliceFastPath
// (struct-field read), walkTypedSliceDisqualifiers and disqualifyAddressOf (escape gate),
// populateSpecialisationStub (specialisation), and (*compiler).sliceElemRegisterKind
// (index/range). All fields are optional; the predicate is nil-safe. Callers without a
// piece of context can pass nil or a partially populated value and the predicate skips
// the corresponding gate. The struct lives next to the predicate so reviewers see the
// unified API in one location.
type kindPromotionContext struct {
	// substitutions maps generic TypeParams to concrete instantiation types; nil for
	// non-generic or non-specialised compilations.
	//
	// Mirrors *compiler.typeSubstitutions.
	substitutions map[*types.TypeParam]types.Type

	// substitutionCache memoises previous substitutions so repeat lookups skip the
	// allocation. Optional; substituteType is nil-safe on the cache.
	substitutionCache map[types.Type]types.Type

	// disqualified is the candidate-name set produced by the escape gate
	// walkTypedSliceDisqualifiers.
	//
	// The predicate refuses promotion when ctx.disqualified[ctx.bindingName] is true. Empty
	// or nil map disables the gate.
	disqualified map[string]bool

	// bindingName is the source-level name being classified (parameter or local var name).
	//
	// Empty when not meaningful (struct field reads, array index expressions). Only
	// consulted when ctx.disqualified is non-empty.
	bindingName string

	// calleeParamPromotions, if non-nil, gates cross-function consistency: a caller cannot
	// promote a value passed into a parameter the callee has already published as
	// non-promoted. Indexed by calleeParamIndex.
	calleeParamPromotions []bool

	// calleeParamIndex selects the index into calleeParamPromotions to consult, when
	// calleeParamPromotions is non-nil.
	calleeParamIndex int
}

// kindForPromotedSlot is the unified typed-bank-promotion predicate.
//
// All five boundary sites consult the same predicate so the kind decision agrees
// end-to-end. The predicate first substitutes t through ctx.substitutions when non-nil so
// specialised generic bodies see their concrete element type, then derives the typed-bank
// candidate via kindForCallSlot (which applies the strict cross-boundary-safety rules).
// It refuses promotion when the candidate is not a typed-slice kind, when
// ctx.disqualified[ctx.bindingName] is set (escape analysis vetoed the binding), or when
// ctx.calleeParamPromotions[ctx.calleeParamIndex] is explicitly false (callee already
// published the parameter as non-promoted). Otherwise it returns the candidate typed-bank
// kind and true.
//
// On every refusal path the returned kind is whatever the type-only verdict produced via
// kindForCallSlot, which is registerGeneral for disqualified slice types and the
// appropriate scalar bank for non-slice types. The bool distinguishes "promoted
// typed-bank (true)" from "fell through (false)". Callers that only care about the kind
// can discard the bool.
//
// Takes t (types.Type) which is the Go type to classify.
// Takes ctx (*kindPromotionContext) which carries optional substitution map, disqualifier
// set, and cross-function escape verdicts. May be nil for sites with no context
// (type-only verdict).
//
// Returns the register kind for the slot and a bool indicating whether the typed-bank
// promotion fired. (registerGeneral, false) when the type alone forbids promotion or any
// gate refused it; the type-only verdict + true when promotion succeeded.
func kindForPromotedSlot(t types.Type, ctx *kindPromotionContext) (registerKind, bool) {
	if ctx != nil && ctx.substitutions != nil {
		t = substituteType(t, ctx.substitutions, ctx.substitutionCache)
	}
	typed := kindForCallSlot(t)
	if !isTypedSliceKind(typed) {
		return typed, false
	}
	if ctx != nil && ctx.bindingName != "" && ctx.disqualified[ctx.bindingName] {
		return registerGeneral, false
	}
	if ctx != nil && ctx.calleeParamPromotions != nil {
		if ctx.calleeParamIndex < 0 || ctx.calleeParamIndex >= len(ctx.calleeParamPromotions) {
			return typed, true
		}
		if !ctx.calleeParamPromotions[ctx.calleeParamIndex] {
			return registerGeneral, false
		}
	}
	return typed, true
}

// kindForTypedSlice returns the typed slice bank for a slice type.
//
// Falls back to registerGeneral when the type has no typed-storage bank. The compiler
// calls this at allocation sites where the value is known to live in the typed slice bank
// (make-typed, typed slice parameter binding) so downstream emission can choose the
// direct opcodes. The mapping covers the five element kinds registerSliceInt /
// registerSliceFloat / registerSliceString / registerSliceBool / registerSliceUint, plus
// registerSliceByte for the compact-stride byte slice bank, routing any compatible typed
// slice through its native bank.
//
// Takes t (types.Type) which is the Go type to classify; non-slice types fall through to
// registerGeneral.
//
// Returns the matching typed-slice register kind.
// Returns registerGeneral when no typed bank applies.
func kindForTypedSlice(t types.Type) registerKind {
	slice, ok := t.Underlying().(*types.Slice)
	if !ok {
		return registerGeneral
	}

	if basic, ok := slice.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.Uint8 {
		return registerSliceByte
	}
	switch kindForType(slice.Elem()) {
	case registerInt:
		return registerSliceInt
	case registerFloat:
		return registerSliceFloat
	case registerString:
		return registerSliceString
	case registerBool:
		return registerSliceBool
	case registerUint:
		return registerSliceUint
	default:
	}
	return registerGeneral
}

// isTypedSliceKind reports whether kind is one of the five typed slice register banks
// (Int / Float / String / Bool / Uint). Used at compile-emit sites that route between
// general-bank reflect storage and the native typed-bank fast-path.
//
// Takes kind (registerKind) which is the kind under test.
//
// Returns true when kind matches one of the typed-slice banks.
func isTypedSliceKind(kind registerKind) bool {
	switch kind {
	case registerSliceInt, registerSliceFloat, registerSliceString, registerSliceBool, registerSliceUint, registerSliceByte:
		return true
	default:
	}
	return false
}

// elementKindForTypedSlice returns the element register kind that the given typed-slice
// bank stores.
//
// For example registerSliceFloat holds registerFloat elements. Returns registerGeneral
// when kind is not a typed-slice bank.
//
// Takes kind (registerKind) which is the typed-slice bank.
//
// Returns the element register kind, or registerGeneral on non-typed-slice inputs.
func elementKindForTypedSlice(kind registerKind) registerKind {
	switch kind {
	case registerSliceInt:
		return registerInt
	case registerSliceFloat:
		return registerFloat
	case registerSliceString:
		return registerString
	case registerSliceBool:
		return registerBool
	case registerSliceUint, registerSliceByte:
		return registerUint
	default:
	}
	return registerGeneral
}

// narrowIntegerBitWidth returns the bit width of a narrow integer type, or 0 for
// full-width and non-integer types.
//
// Recognises int8/int16/int32 and uint8/uint16/uint32. The compiler uses this to emit
// opTruncateNarrow after arithmetic on narrow integer kinds so the register value matches
// Go's modular wrap semantics.
//
// Takes t (types.Type) which is the Go type to inspect.
//
// Returns the bit width when truncation is needed, or 0 otherwise.
func narrowIntegerBitWidth(t types.Type) uint8 {
	if t == nil {
		return 0
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return 0
	}
	switch basic.Kind() {
	case types.Int8, types.Uint8:
		return narrowIntegerBitWidth8
	case types.Int16, types.Uint16:
		return narrowIntegerBitWidth16
	case types.Int32, types.Uint32:
		return narrowIntegerBitWidth32
	default:
		return 0
	}
}

// kindForBasic maps a types.BasicKind to a registerKind.
//
// Takes k (types.BasicKind) which is the basic type kind to map.
//
// Returns the corresponding register kind.
func kindForBasic(k types.BasicKind) registerKind {
	switch k {
	case types.Bool, types.UntypedBool:
		return registerBool

	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.UntypedInt, types.UntypedRune:
		return registerInt

	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr:
		return registerUint

	case types.Float32, types.Float64, types.UntypedFloat:
		return registerFloat

	case types.String, types.UntypedString:
		return registerString

	case types.Complex64, types.Complex128, types.UntypedComplex:
		return registerComplex

	default:
		return registerGeneral
	}
}
