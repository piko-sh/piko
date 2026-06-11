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
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// fastPathDispatcherTag labels recordFastPathTypeMismatch records so callers can
	// attribute mismatches to the fastpath dispatch layer.
	fastPathDispatcherTag = "fastpath dispatcher"

	// digestBytesMD5 is the byte width of an MD5 digest (also the UUID length). Used by
	// fpDispatchBytesArr16 to size the arena allocation and as the [16]byte type parameter.
	digestBytesMD5 = 16

	// digestBytesSHA1 is the byte width of a SHA-1 digest. Used by fpDispatchBytesArr20.
	digestBytesSHA1 = 20

	// digestBytesSHA256 is the byte width of a SHA-256 digest. Used by fpDispatchBytesArr32.
	digestBytesSHA256 = 32

	// digestBytesSHA512 is the byte width of a SHA-512 digest. Used by fpDispatchBytesArr64.
	digestBytesSHA512 = 64
)

const (
	// fastPathTagNone indicates that no fast-path dispatch is available for this call site
	// after classification has been attempted. The zero value (unprobed) is implicit - any
	// call site starts at 0 before classification.
	fastPathTagNone nativeFastPathTag = iota + 1

	// fastPathTagStringString represents the fast-path tag for native functions with the
	// signature func(string) string.
	fastPathTagStringString

	// fastPathTagStringInt represents the fast-path tag for native functions with the
	// signature func(string) int.
	fastPathTagStringInt

	// fastPathTagStringBool represents the fast-path tag for native functions with the
	// signature func(string) bool.
	fastPathTagStringBool

	// fastPathTagStringRuneBool represents the fast-path tag for native functions with the
	// signature func(string, int32) bool.
	fastPathTagStringRuneBool

	// fastPathTagStringRuneInt represents the fast-path tag for native functions with the
	// signature func(string, int32) int.
	fastPathTagStringRuneInt

	// fastPathTagString2Bool represents the fast-path tag for native functions with the
	// signature func(string, string) bool.
	fastPathTagString2Bool

	// fastPathTagString2String represents the fast-path tag for native functions with the
	// signature func(string, string) string.
	fastPathTagString2String

	// fastPathTagString2Int represents the fast-path tag for native functions with the
	// signature func(string, string) int.
	fastPathTagString2Int

	// fastPathTagString3String represents the fast-path tag for native functions with the
	// signature func(string, string, string) string.
	fastPathTagString3String

	// fastPathTagIntString represents the fast-path tag for native functions with the
	// signature func(int) string.
	fastPathTagIntString

	// fastPathTagIntInt represents the fast-path tag for native functions with the signature
	// func(int) int.
	fastPathTagIntInt

	// fastPathTagIntBool represents the fast-path tag for native functions with the
	// signature func(int) bool.
	fastPathTagIntBool

	// fastPathTagInt2Int represents the fast-path tag for native functions with the
	// signature func(int, int) int.
	fastPathTagInt2Int

	// fastPathTagInt2Bool represents the fast-path tag for native functions with the
	// signature func(int, int) bool.
	fastPathTagInt2Bool

	// fastPathTagInt2String represents the fast-path tag for native functions with the
	// signature func(int, int) string.
	fastPathTagInt2String

	// fastPathTagInt64IntString represents the fast-path tag for native functions with the
	// signature func(int64, int) string.
	fastPathTagInt64IntString

	// fastPathTagStringIntError represents the fast-path tag for native functions with the
	// signature func(string) (int, error).
	fastPathTagStringIntError

	// fastPathTagFloat64Float64 represents the fast-path tag for native functions with the
	// signature func(float64) float64.
	fastPathTagFloat64Float64

	// fastPathTagFloat642Float64 represents the fast-path tag for native functions with the
	// signature func(float64, float64) float64.
	fastPathTagFloat642Float64

	// fastPathTagAnyBool represents the fast-path tag for native functions with the
	// signature func(any) bool.
	fastPathTagAnyBool

	// fastPathTagAnyString represents the fast-path tag for native functions with the
	// signature func(any) string.
	fastPathTagAnyString

	// fastPathTagAnyInt represents the fast-path tag for native functions with the signature
	// func(any) int.
	fastPathTagAnyInt

	// fastPathTagAnyInt64 represents the fast-path tag for native functions with the
	// signature func(any) int64.
	fastPathTagAnyInt64

	// fastPathTagAnyFloat64 represents the fast-path tag for native functions with the
	// signature func(any) float64.
	fastPathTagAnyFloat64

	// fastPathTagAny2Any represents the fast-path tag for native functions with the
	// signature func(any, any) any.
	fastPathTagAny2Any

	// fastPathTagRetString represents the fast-path tag for native functions with the
	// signature func() string.
	fastPathTagRetString

	// fastPathTagRetBool represents the fast-path tag for native functions with the
	// signature func() bool.
	fastPathTagRetBool

	// fastPathTagRetInt represents the fast-path tag for native functions with the signature
	// func() int.
	fastPathTagRetInt

	// fastPathTagRetInt64 represents the fast-path tag for native functions with the
	// signature func() int64.
	fastPathTagRetInt64

	// fastPathTagRetFloat64 represents the fast-path tag for native functions with the
	// signature func() float64.
	fastPathTagRetFloat64

	// fastPathTagRetError represents the fast-path tag for native functions with the
	// signature func() error.
	fastPathTagRetError

	// fastPathTagVoid represents the fast-path tag for native functions with the signature
	// func().
	fastPathTagVoid

	// fastPathTagVoidString represents the fast-path tag for native functions with the
	// signature func(string).
	fastPathTagVoidString

	// fastPathTagVoidInt represents the fast-path tag for native functions with the
	// signature func(int).
	fastPathTagVoidInt

	// fastPathTagVoidInt64 represents the fast-path tag for native functions with the
	// signature func(int64).
	fastPathTagVoidInt64

	// fastPathTagVoidBool represents the fast-path tag for native functions with the
	// signature func(bool).
	fastPathTagVoidBool

	// fastPathTagVoidString2 represents the fast-path tag for native functions with the
	// signature func(string, string).
	fastPathTagVoidString2

	// fastPathTagStringError represents the fast-path tag for native functions with the
	// signature func(string) error.
	fastPathTagStringError

	// fastPathTagSprintfString represents the fast-path tag for native functions with the
	// signature func(string, ...any) string.
	fastPathTagSprintfString

	// fastPathTagSprintfError represents the fast-path tag for native functions with the
	// signature func(string, ...any) error.
	fastPathTagSprintfError

	// fastPathTagSprintVarargs represents the fast-path tag for native functions with the
	// signature func(...any) string.
	fastPathTagSprintVarargs

	// fastPathTagBytesArr32 represents the fast-path tag for native functions with the
	// signature func([]byte) [32]byte. Covers SHA-256 and similar fixed-output hashes; the
	// 32-byte digest is bump-allocated into the arena byte slab and wrapped via unsafeNewAt,
	// bypassing reflect.Call.
	fastPathTagBytesArr32

	// fastPathTagBytesArr16 represents the fast-path tag for native functions with the
	// signature func([]byte) [16]byte. Covers MD5 (md5.Sum) and UUID-shape hashes via the
	// same arena-routed mechanism as fastPathTagBytesArr32.
	fastPathTagBytesArr16

	// fastPathTagBytesArr20 represents the fast-path tag for native functions with the
	// signature func([]byte) [20]byte. Covers SHA-1 (sha1.Sum) via the same arena-routed
	// mechanism.
	fastPathTagBytesArr20

	// fastPathTagBytesArr64 represents the fast-path tag for native functions with the
	// signature func([]byte) [64]byte. Covers SHA-512 (sha512.Sum512) via the same
	// arena-routed mechanism.
	fastPathTagBytesArr64

	// pikoIDFieldPrefix marks struct fields piko's compiler injects on
	// reflect.StructOf-synthesised types to flag them as piko-managed.
	pikoIDFieldPrefix = "_pikoID_"
)

// nativeFastPathEntry is the immutable per-site classification cache.
//
// callSites live in a CompiledFunction shared across the per-goroutine VMs a `go`
// statement spawns, so the entry is published through an atomic pointer rather than
// mutated field by field: a plain write of the two-word fn interface could be read torn
// by a sibling goroutine, splicing one value's type word onto another's data word. Each
// refresh stores a fresh entry; an entry is never mutated after publication.
type nativeFastPathEntry struct {
	// fn is the extracted native function value to dispatch, or nativeFastPathNone when
	// classification found no fast path.
	fn any

	// receiverAddr is the address of the method receiver fn was bound to, used to invalidate
	// the cache when the same method is called on a different receiver. Zero for non-method
	// sites.
	receiverAddr uintptr

	// tag selects which fast-path dispatcher handles fn.
	tag nativeFastPathTag
}

var (
	// nativeFastPathNone is the sentinel value indicating that a call site has no fast-path
	// specialisation available.
	nativeFastPathNone = &struct{}{}

	// nativeFastPathNoneEntry is the shared immutable entry stored at a call site once
	// classification finds no fast path. Sharing one instance avoids an allocation each time
	// a non-specialised site is probed.
	nativeFastPathNoneEntry = &nativeFastPathEntry{fn: nativeFastPathNone}

	// fastPathTagByType maps concrete function reflect.Types to their corresponding
	// fast-path tag for O(1) classification.
	fastPathTagByType = map[reflect.Type]nativeFastPathTag{
		reflect.TypeFor[func(string) string]():                 fastPathTagStringString,
		reflect.TypeFor[func(string) int]():                    fastPathTagStringInt,
		reflect.TypeFor[func(string) bool]():                   fastPathTagStringBool,
		reflect.TypeFor[func(string, int32) bool]():            fastPathTagStringRuneBool,
		reflect.TypeFor[func(string, int32) int]():             fastPathTagStringRuneInt,
		reflect.TypeFor[func(string, string) bool]():           fastPathTagString2Bool,
		reflect.TypeFor[func(string, string) string]():         fastPathTagString2String,
		reflect.TypeFor[func(string, string) int]():            fastPathTagString2Int,
		reflect.TypeFor[func(string, string, string) string](): fastPathTagString3String,
		reflect.TypeFor[func(int) string]():                    fastPathTagIntString,
		reflect.TypeFor[func(int) int]():                       fastPathTagIntInt,
		reflect.TypeFor[func(int) bool]():                      fastPathTagIntBool,
		reflect.TypeFor[func(int, int) int]():                  fastPathTagInt2Int,
		reflect.TypeFor[func(int, int) bool]():                 fastPathTagInt2Bool,
		reflect.TypeFor[func(int, int) string]():               fastPathTagInt2String,
		reflect.TypeFor[func(int64, int) string]():             fastPathTagInt64IntString,
		reflect.TypeFor[func(string) (int, error)]():           fastPathTagStringIntError,
		reflect.TypeFor[func(float64) float64]():               fastPathTagFloat64Float64,
		reflect.TypeFor[func(float64, float64) float64]():      fastPathTagFloat642Float64,
		reflect.TypeFor[func(any) bool]():                      fastPathTagAnyBool,
		reflect.TypeFor[func(any) string]():                    fastPathTagAnyString,
		reflect.TypeFor[func(any) int]():                       fastPathTagAnyInt,
		reflect.TypeFor[func(any) int64]():                     fastPathTagAnyInt64,
		reflect.TypeFor[func(any) float64]():                   fastPathTagAnyFloat64,
		reflect.TypeFor[func(any, any) any]():                  fastPathTagAny2Any,
		reflect.TypeFor[func() string]():                       fastPathTagRetString,
		reflect.TypeFor[func() bool]():                         fastPathTagRetBool,
		reflect.TypeFor[func() int]():                          fastPathTagRetInt,
		reflect.TypeFor[func() int64]():                        fastPathTagRetInt64,
		reflect.TypeFor[func() float64]():                      fastPathTagRetFloat64,
		reflect.TypeFor[func() error]():                        fastPathTagRetError,
		reflect.TypeFor[func()]():                              fastPathTagVoid,
		reflect.TypeFor[func(string)]():                        fastPathTagVoidString,
		reflect.TypeFor[func(int)]():                           fastPathTagVoidInt,
		reflect.TypeFor[func(int64)]():                         fastPathTagVoidInt64,
		reflect.TypeFor[func(bool)]():                          fastPathTagVoidBool,
		reflect.TypeFor[func(string, string)]():                fastPathTagVoidString2,
		reflect.TypeFor[func(string) error]():                  fastPathTagStringError,
		reflect.TypeFor[func(string, ...any) string]():         fastPathTagSprintfString,
		reflect.TypeFor[func(string, ...any) error]():          fastPathTagSprintfError,
		reflect.TypeFor[func(...any) string]():                 fastPathTagSprintVarargs,
		reflect.TypeFor[func([]byte) [32]byte]():               fastPathTagBytesArr32,
		reflect.TypeFor[func([]byte) [16]byte]():               fastPathTagBytesArr16,
		reflect.TypeFor[func([]byte) [20]byte]():               fastPathTagBytesArr20,
		reflect.TypeFor[func([]byte) [64]byte]():               fastPathTagBytesArr64,
	}

	// fastPathDispatchTable is an array of dispatch functions indexed by nativeFastPathTag.
	// Populated at init time.
	fastPathDispatchTable [fastPathTagBytesArr64 + 1]fastPathDispatcher

	// bytesArr32ABIType caches the *abi.Type pointer for [32]byte so the fast-path
	// dispatcher can construct the result reflect.Value via unsafeNewAt without
	// re-extracting the ABI type on every call.
	bytesArr32ABIType = reflectValueABIType(reflect.TypeFor[[32]byte]())

	// bytesArr16ABIType caches the *abi.Type pointer for [16]byte (MD5 / UUID byte size).
	bytesArr16ABIType = reflectValueABIType(reflect.TypeFor[[16]byte]())

	// bytesArr20ABIType caches the *abi.Type pointer for [20]byte (SHA-1 digest size).
	bytesArr20ABIType = reflectValueABIType(reflect.TypeFor[[20]byte]())

	// bytesArr64ABIType caches the *abi.Type pointer for [64]byte (SHA-512 digest size).
	bytesArr64ABIType = reflectValueABIType(reflect.TypeFor[[64]byte]())
)

// nativeFastPathTag identifies which fast-path case matched so that subsequent calls can
// dispatch via a uint8 jump table instead of the full interface type switch.
type nativeFastPathTag uint8

// fastPathDispatcher is the signature for individual fast-path dispatch functions, each
// handling exactly one tag case.
type fastPathDispatcher func(vm *VM, cached any, site *callSite, registers *Registers)

// reuseVarArgsBuf returns a []any slice of length n, reusing the pre-allocated buffer on
// the callSite when possible.
//
// Takes n (int) which is the required slice length.
//
// Returns a []any slice of the requested length, either resliced from the existing buffer
// or freshly allocated if the buffer capacity is insufficient.
func (cs *callSite) reuseVarArgsBuf(n int) []any {
	if cap(cs.variadicArgumentsBuffer) >= n {
		return cs.variadicArgumentsBuffer[:n]
	}

	cs.variadicArgumentsBuffer = make([]any, n)

	return cs.variadicArgumentsBuffer
}

// classifyNativeFastPath determines the fast-path tag for a given function value by
// looking up its reflect.Type in the dispatch map.
//
// Takes v (any) which is the native function value to classify.
//
// Returns the matching nativeFastPathTag, or fastPathTagNone if no fast-path
// specialisation exists for the function's type signature.
func classifyNativeFastPath(v any) nativeFastPathTag {
	tag, ok := fastPathTagByType[reflect.TypeOf(v)]
	if !ok {
		return fastPathTagNone
	}

	return tag
}

// tryNativeFastPath attempts to call a native function via a direct type assertion on the
// already-extracted function value, bypassing reflect.Value.Call().
//
// The caller is responsible for extracting the function value (via
// reflectedFunction.Interface()) and passing it as v. For handleCallNative, the extracted
// value is cached on the callSite for subsequent calls, eliminating repeated Interface()
// allocations. For handleCall (closure dispatch), the value is extracted fresh each time
// because the underlying function may change between calls.
//
// Takes site (*callSite) which is the call site metadata including argument and return
// register locations.
// Takes v (any) which is the native function value to dispatch.
// Takes registers (*Registers) which is the VM register file to read arguments from and
// write results to.
//
// Returns true and the matched tag if the fast path was taken, or false and
// fastPathTagNone if no fast-path specialisation is available. The third return is the
// recovered panic from the native call (e.g. sync.Mutex.Unlock on an unlocked mutex), nil
// if none.
func tryNativeFastPath(vm *VM, site *callSite, v any, registers *Registers) (bool, nativeFastPathTag, any) {
	tag := classifyNativeFastPath(v)
	if tag == fastPathTagNone {
		atomic.StorePointer(&site.nativeFastPath, unsafe.Pointer(nativeFastPathNoneEntry))
		return false, fastPathTagNone, nil
	}

	if site.isEllipsisSpread {
		return false, fastPathTagNone, nil
	}

	if siteArgsRequireInterfaceAdapter(vm, site, registers) {
		return false, fastPathTagNone, nil
	}

	panicValue := dispatchNativeFastPathTagged(vm, tag, v, site, registers)

	return true, tag, panicValue
}

// siteArgsRequireInterfaceAdapter reports whether any of the site's general-bank
// arguments is a piko-synthesised value that needs to be wrapped in an adapter
// (pikoErrorAdapter, pikoStringerAdapter, ...) before it can satisfy the native
// function's interface parameter. Such args bypass the fast path so the slow
// handleCallNativeReflect path runs coerceReflectArgument, which builds the adapter.
//
// Two conditions force the slow path. The first is when the parameter slot at index i is
// an interface kind (recorded in site.parameterInterfaceFlags at compile time) and the
// argument value is piko-synthesised, so the native function's reflect.Type.Implements
// check sees the right method set. The second is when the argument's static or dynamic
// type itself has a registered adapter (Error, String, MarshalJSON); this covers `any` /
// `interface{}` parameters whose callee still consults Implements at runtime, fmt's verbs
// being the canonical example. The first check is cheap (O(1) bool index plus the
// `_pikoID_` sentinel field walk) and non-piko args never reach the dynamic check.
//
// Takes vm (*VM) which owns the method table consulted for adapter eligibility.
// Takes site (*callSite) which holds the per-parameter interface flags and per-arg static
// type names.
// Takes registers (*Registers) which holds the live argument values.
//
// Returns true when at least one argument needs adapter wrapping.
func siteArgsRequireInterfaceAdapter(vm *VM, site *callSite, registers *Registers) bool {
	for i, argument := range site.arguments {
		if parameterSlotIsInterface(site, i) {
			if argument.kind == registerGeneral {
				value := registers.general[argument.register]
				if value.IsValid() && argumentIsPikoSynthesised(value) {
					return true
				}
			}
		}
		if argStaticTypeNameRequiresAdapter(vm, site, i) {
			return true
		}
		if argDynamicValueRequiresAdapter(vm, argument, registers) {
			return true
		}
	}
	return false
}

// parameterSlotIsInterface reports whether the i-th fixed parameter of the callee has
// interface kind. Variadic tails are not recorded in parameterInterfaceFlags - they're
// handled separately via the `_pikoID_` sentinel check in the fast-path `any`-tail
// dispatchers.
//
// Takes site (*callSite) which carries the compile-time flag table.
// Takes argumentIndex (int) which is the position to inspect.
//
// Returns false when no flag table exists, the position is past the table, or the slot is
// concrete.
func parameterSlotIsInterface(site *callSite, argumentIndex int) bool {
	if argumentIndex >= len(site.parameterInterfaceFlags) {
		return false
	}
	return site.parameterInterfaceFlags[argumentIndex]
}

// argumentIsPikoSynthesised peels one layer of interface boxing and returns true when the
// underlying reflect.Type carries piko's `_pikoID_` sentinel field. Used by the fast-path
// interface-slot check to decide whether to fall through to the slow path's adapter
// builder.
//
// Takes value (reflect.Value) which is the live argument value.
//
// Returns true when the value's reflect.Type is piko-synthesised.
func argumentIsPikoSynthesised(value reflect.Value) bool {
	probe := value
	if probe.Kind() == reflect.Interface && !probe.IsNil() {
		probe = probe.Elem()
	}
	if !probe.IsValid() {
		return false
	}
	return isPikoSynthesisedReflectType(probe.Type())
}

// argStaticTypeNameRequiresAdapter checks the compiler-recorded static type name for
// argument i against the interface-adapter method table.
//
// Takes vm (*VM) which is the active interpreter instance.
// Takes site (*callSite) which carries the compiler-recorded static names.
// Takes argumentIndex (int) which is the zero-based argument position.
//
// Returns true when the static name is registered as needing the adapter; false when no
// static name is recorded at the index.
func argStaticTypeNameRequiresAdapter(vm *VM, site *callSite, argumentIndex int) bool {
	if argumentIndex >= len(site.argumentStaticTypeNames) {
		return false
	}
	staticName := site.argumentStaticTypeNames[argumentIndex]
	if staticName == "" {
		return false
	}
	return typeNameHasInterfaceAdapter(vm, staticName)
}

// argDynamicValueRequiresAdapter inspects the live general-bank register for a
// piko-synthesised reflect.Type whose source-level name has a registered adapter method.
//
// Takes vm (*VM) which is the active interpreter instance.
// Takes argument (varLocation) which is the location of the argument register to inspect.
// Takes registers (*Registers) which is the active register bank containing the value.
//
// Returns true when the dynamic value resolves to a piko-synthesised type with a
// registered adapter; false for non-general-bank arguments or non-synthesised types.
func argDynamicValueRequiresAdapter(vm *VM, argument varLocation, registers *Registers) bool {
	if argument.kind != registerGeneral {
		return false
	}
	value := registers.general[argument.register]
	if !value.IsValid() {
		return false
	}
	probe := value
	if probe.Kind() == reflect.Interface && !probe.IsNil() {
		probe = probe.Elem()
	}
	if !isPikoSynthesisedReflectType(probe.Type()) {
		return false
	}
	typeName, ok := pikoTypeName(vm, probe)
	if !ok {
		return false
	}
	return typeNameHasUsableInterfaceAdapter(vm, typeName, probe.Kind() == reflect.Pointer)
}

// typeNameHasUsableInterfaceAdapter reports whether the named source-level type has at
// least one registered method that is usable given the value's pointer/value shape. Go's
// method-set rule means a value `T` only has its value-receiver methods, while a pointer
// `*T` has both value- and pointer-receiver methods.
//
// Without this gate, a value typed `opaqueError` whose only method is `(o *opaqueError)
// Error()` would skip the fast path, reach tryBuildInterfaceAdapter, and be rejected by
// methodReceiverSatisfiesValue, but the rejection costs the fast-path wrap that would
// otherwise hide the `_pikoID_` sentinel field from fmt's default printer. Keeping the
// fast path lets wrapPikoSynthesisedFmtArg apply the pikoFmtValue wrapper that renders
// the struct without the sentinel.
//
// Takes vm (*VM) which owns the method table.
// Takes typeName (string) which is the source-level type name.
// Takes isPointerValue (bool) which is true when the runtime value is a pointer (`*T`);
// false when it is the value type (`T`).
//
// Returns true when at least one method satisfies the receiver-kind rule for the value
// shape.
func typeNameHasUsableInterfaceAdapter(vm *VM, typeName string, isPointerValue bool) bool {
	if vm == nil || typeName == "" {
		return false
	}
	for _, methodName := range []string{".Error", ".String", ".MarshalJSON"} {
		methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+methodName)
		if !ok {
			continue
		}
		callee, _, ok := resolveAdapterCallee(vm, methodRoot, methodIndex)
		if !ok || callee == nil {
			continue
		}
		if callee.isPointerReceiver && !isPointerValue {
			continue
		}
		return true
	}
	return false
}

// adaptArgForNativeAny prepares an `any`-typed argument for fast-path calls.
//
// The fast-path callers (fpDispatchAnyX, fpDispatchAny2Any) need the adaptation when the
// argument is a piko-synthesised struct and the callee is a genuinely native Go function
// (not a piko-side reflect.MakeFunc closure representing a method expression or
// interpreted closure). In that case piko-aware fmt/adapter wrapping is applied so the
// native callee sees the value through the right interface (fmt.Stringer, fmt.Formatter,
// error, etc.). For piko-side MakeFunc closures the argument passes through unchanged:
// the closure expects the concrete underlying type at its receiver slot, and substituting
// a wrapper would break receiver identity for pointer-receiver methods or fail type
// assertions inside the interpreted body. Callee classification piggybacks on the
// reflectMakeFuncStubPointer sentinel via isPikoMakeFuncClosure, a single pointer compare
// on the already-extracted cached function.
//
// Takes vm (*VM) which is the active interpreter.
// Takes cached (any) which is the unwrapped native function value the dispatcher already
// type-asserted.
// Takes raw (any) which is the just-read argument from readAnyArg.
//
// Returns the argument unchanged when no wrapping is needed; a pikoFmtValue or adapter
// when the wrap applies.
func adaptArgForNativeAny(vm *VM, cached any, raw any) any {
	if raw == nil {
		return nil
	}
	if calleeIsPikoMakeFuncClosure(cached) {
		return raw
	}
	return wrapPikoSynthesisedFmtArg(vm, raw)
}

// readNativeAnyArg reads one argument destined for a native interface ({}) parameter and
// re-clothes it with its source-level named type before adapting.
//
// Takes vm (*VM) which provides the symbol registry for the type restore.
// Takes cached (any) which is the native callee, used by adaptArgForNativeAny.
// Takes site (*callSite) which carries the per-argument static type strings.
// Takes registers (*Registers) which holds the source value.
// Takes argumentIndex (int) which selects the argument and its static type string.
//
// Returns the prepared argument value ready to pass to the native function.
func readNativeAnyArg(vm *VM, cached any, site *callSite, registers *Registers, argumentIndex int) any {
	raw := readAnyArg(registers, site.arguments[argumentIndex])
	if argumentIndex < len(site.argumentStaticTypeStrings) {
		raw = restoreNamedTypeForFmt(vm, raw, site.argumentStaticTypeStrings[argumentIndex])
	}
	return adaptArgForNativeAny(vm, cached, raw)
}

// calleeIsPikoMakeFuncClosure reports whether the cached native function value is
// actually a piko-side reflect.MakeFunc closure. All reflect.MakeFunc closures share one
// trampoline code pointer (captured at init in reflectMakeFuncStubPointer); native Go
// functions never report that pointer, so a single uintptr compare reliably distinguishes
// the two kinds.
//
// Takes cached (any) which is the dispatcher's cached function.
//
// Returns true when cached is a reflect.MakeFunc result.
func calleeIsPikoMakeFuncClosure(cached any) bool {
	if cached == nil {
		return false
	}
	value := reflect.ValueOf(cached)
	if value.Kind() != reflect.Func {
		return false
	}
	return value.Pointer() == reflectMakeFuncStubPointer
}

// typeNameHasInterfaceAdapter reports whether the named source-level type has a
// registered method that one of the piko interface adapters knows how to bridge (Error /
// String / MarshalJSON). When true, the caller bypasses the fast path so the slow path
// can build the adapter via tryBuildInterfaceAdapter.
//
// Takes vm (*VM) which owns the method table.
// Takes typeName (string) which is the source-level type name.
//
// Returns true when typeName has at least one bridgeable method.
func typeNameHasInterfaceAdapter(vm *VM, typeName string) bool {
	if vm == nil || typeName == "" {
		return false
	}
	if _, _, ok := lookupAdapterMethod(vm, typeName+".Error"); ok {
		return true
	}
	if _, _, ok := lookupAdapterMethod(vm, typeName+".String"); ok {
		return true
	}
	if _, _, ok := lookupAdapterMethod(vm, typeName+".MarshalJSON"); ok {
		return true
	}
	return false
}

var (
	// pikoSynthesisedTypeCache memoises type classification results.
	//
	// Keyed by reflect.Type and consulted by isPikoSynthesisedReflectType. reflect.Type
	// values are interned and stable, so the cache is bounded by the program's distinct
	// struct types. It collapses a per-native-call, per-argument field walk (over types such
	// as http.Request with many fields) into a single map load. sync.Map suits the
	// read-mostly hot path: entries are written once on first observation and read on every
	// subsequent native call.
	//
	//nolint:gochecknoglobals // process-wide immutable-after-first-write type cache
	pikoSynthesisedTypeCache sync.Map
)

// isPikoSynthesisedReflectType reports whether t is piko-synthesised.
//
// Detects structs the compiler created via reflect.StructOf with the `_pikoID_` sentinel
// field. Used to decide whether a value needs interface-adapter wrapping at native-call
// boundaries. The field walk is performed once per type and memoised, because the same
// handful of types recur across every native call.
//
// Takes t (reflect.Type) which is the candidate reflect type (pointers are unwrapped
// before checking).
//
// Returns true when t (or its pointee) carries a `_pikoID_`-prefixed field.
func isPikoSynthesisedReflectType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	if cached, ok := pikoSynthesisedTypeCache.Load(t); ok {
		if result, isBool := cached.(bool); isBool {
			return result
		}
	}
	result := false
	for field := range t.Fields() {
		if strings.HasPrefix(field.Name, pikoIDFieldPrefix) {
			result = true
			break
		}
	}
	pikoSynthesisedTypeCache.Store(t, result)
	return result
}

// dispatchNativeFastPathTagged dispatches a cached native call using the pre-resolved tag
// via an array lookup, avoiding the sequential itab comparisons of a full type switch.
// The dispatch is wrapped in a recover boundary so panics from native functions (sync.*
// misuse, channel ops on closed channels, etc.) become recoverable from interpreted
// defer/recover.
//
// Takes tag (nativeFastPathTag) which is the pre-classified fast-path tag.
// Takes cached (any) which is the native function value extracted via Interface().
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file to read arguments from and
// write results to.
//
// Returns the recovered panic value, or nil if the dispatch completed without panicking.
func dispatchNativeFastPathTagged(vm *VM, tag nativeFastPathTag, cached any, site *callSite, registers *Registers) (panicValue any) {
	defer func() {
		if r := recover(); r != nil {
			panicValue = r
		}
	}()
	if int(tag) < len(fastPathDispatchTable) {
		if d := fastPathDispatchTable[tag]; d != nil {
			d(vm, cached, site, registers)
		}
	}
	return nil
}

// fpDispatchStringString dispatches a native function with the signature func(string)
// string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchStringInt dispatches a native function with the signature func(string) int
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(readStringArg(registers, site.arguments[0])))
}

// fpDispatchStringBool dispatches a native function with the signature func(string) bool
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchStringRuneBool dispatches a native function with the signature func(string,
// int32) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringRuneBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, int32) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		safeconv.Int64ToInt32(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchStringRuneInt dispatches a native function with the signature func(string,
// int32) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringRuneInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, int32) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		readStringArg(registers, site.arguments[0]),
		safeconv.Int64ToInt32(readIntArg(registers, site.arguments[1])),
	))
}

// fpDispatchString2Bool dispatches a native function with the signature func(string,
// string) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2Bool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	)
}

// fpDispatchString2String dispatches a native function with the signature func(string,
// string) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2String(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	)
}

// fpDispatchString2Int dispatches a native function with the signature func(string,
// string) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2Int(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	))
}

// fpDispatchString3String dispatches a native function with the signature func(string,
// string, string) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString3String(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string, string) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
		readStringArg(registers, site.arguments[2]),
	)
}

// fpDispatchIntString dispatches a native function with the signature func(int) string
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchIntInt dispatches a native function with the signature func(int) int via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(int(readIntArg(registers, site.arguments[0]))))
}

// fpDispatchIntBool dispatches a native function with the signature func(int) bool via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchInt2Int dispatches a native function with the signature func(int, int) int
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2Int(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	))
}

// fpDispatchInt2Bool dispatches a native function with the signature func(int, int) bool
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2Bool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchInt2String dispatches a native function with the signature func(int, int)
// string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2String(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchInt64IntString dispatches a native function with the signature func(int64,
// int) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt64IntString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int64, int) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(
		readIntArg(registers, site.arguments[0]),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchStringIntError dispatches a native function with the signature func(string)
// (int, error) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringIntError(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) (int, error))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	value, err := f(readStringArg(registers, site.arguments[0]))
	registers.ints[site.returns[0].register] = int64(value)

	if len(site.returns) > 1 {
		if err != nil {
			registers.general[site.returns[1].register] = reflect.ValueOf(err)
		} else {
			registers.general[site.returns[1].register] = reflect.Value{}
		}
	}
}

// fpDispatchFloat64Float64 dispatches a native function with the signature func(float64)
// float64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchFloat64Float64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(float64) float64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.floats[site.returns[0].register] = f(readFloatArg(registers, site.arguments[0]))
}

// fpDispatchFloat642Float64 dispatches a native function with the signature func(float64,
// float64) float64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchFloat642Float64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(float64, float64) float64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.floats[site.returns[0].register] = f(
		readFloatArg(registers, site.arguments[0]),
		readFloatArg(registers, site.arguments[1]),
	)
}

// fpDispatchAnyBool dispatches a native function with the signature func(any) bool via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f(readNativeAnyArg(vm, cached, site, registers, 0))
}

// fpDispatchAnyString dispatches a native function with the signature func(any) string
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f(readNativeAnyArg(vm, cached, site, registers, 0))
}

// fpDispatchAnyInt dispatches a native function with the signature func(any) int via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f(readNativeAnyArg(vm, cached, site, registers, 0)))
}

// fpDispatchAnyInt64 dispatches a native function with the signature func(any) int64 via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyInt64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) int64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = f(readNativeAnyArg(vm, cached, site, registers, 0))
}

// fpDispatchAnyFloat64 dispatches a native function with the signature func(any) float64
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyFloat64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) float64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.floats[site.returns[0].register] = f(readNativeAnyArg(vm, cached, site, registers, 0))
}

// fpDispatchAny2Any dispatches a native function with the signature func(any, any) any
// via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAny2Any(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any, any) any)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	result := f(readNativeAnyArg(vm, cached, site, registers, 0), readNativeAnyArg(vm, cached, site, registers, 1))
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchRetString dispatches a native function with the signature func() string via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.strings[site.returns[0].register] = f()
}

// fpDispatchRetBool dispatches a native function with the signature func() bool via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() bool)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.bools[site.returns[0].register] = f()
}

// fpDispatchRetInt dispatches a native function with the signature func() int via direct
// type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() int)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = int64(f())
}

// fpDispatchRetInt64 dispatches a native function with the signature func() int64 via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetInt64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() int64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.ints[site.returns[0].register] = f()
}

// fpDispatchRetFloat64 dispatches a native function with the signature func() float64 via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetFloat64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() float64)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	registers.floats[site.returns[0].register] = f()
}

// fpDispatchRetError dispatches a native function with the signature func() error via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetError(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() error)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	result := f()
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchVoid dispatches a native function with the signature func() via direct type
// assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
func fpDispatchVoid(vm *VM, cached any, _ *callSite, _ *Registers) {
	f, ok := cached.(func())
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f()
}

// fpDispatchVoidString dispatches a native function with the signature func(string) via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchVoidInt dispatches a native function with the signature func(int) via direct
// type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidInt(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchVoidInt64 dispatches a native function with the signature func(int64) via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidInt64(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int64))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f(readIntArg(registers, site.arguments[0]))
}

// fpDispatchVoidBool dispatches a native function with the signature func(bool) via
// direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidBool(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(bool))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f(readBoolArg(registers, site.arguments[0]))
}

// fpDispatchVoidString2 dispatches a native function with the signature func(string,
// string) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidString2(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string))
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	f(readStringArg(registers, site.arguments[0]), readStringArg(registers, site.arguments[1]))
}

// fpDispatchStringError dispatches a native function with the signature func(string)
// error via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringError(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) error)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	result := f(readStringArg(registers, site.arguments[0]))
	if len(site.returns) > 0 {
		if result != nil {
			registers.general[site.returns[0].register] = reflect.ValueOf(result)
		} else {
			registers.general[site.returns[0].register] = reflect.Value{}
		}
	}
}

// fpDispatchSprintfString dispatches a native function with the signature func(string,
// ...any) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintfString(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, ...any) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	format := readStringArg(registers, site.arguments[0])
	nVarArgs := len(site.arguments) - 1
	varArgs := site.reuseVarArgsBuf(nVarArgs)

	for i := range nVarArgs {
		raw := readAnyArg(registers, site.arguments[i+1])
		argIdx := i + 1
		if argIdx < len(site.argumentStaticTypeStrings) {
			raw = restoreNamedTypeForFmt(vm, raw, site.argumentStaticTypeStrings[argIdx])
		}
		varArgs[i] = wrapPikoSynthesisedFmtArg(vm, raw)
	}

	if rewrittenFormat, rewrittenArgs, intercepted := interceptFmtFormat(site, 1, format, varArgs); intercepted {
		format = rewrittenFormat
		varArgs = rewrittenArgs
	}

	registers.strings[site.returns[0].register] = f(format, varArgs...)
}

// fpDispatchSprintfError dispatches a native function with the signature func(string,
// ...any) error via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintfError(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, ...any) error)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	format := readStringArg(registers, site.arguments[0])
	nVarArgs := len(site.arguments) - 1
	varArgs := site.reuseVarArgsBuf(nVarArgs)

	for i := range nVarArgs {
		raw := readAnyArg(registers, site.arguments[i+1])
		argIdx := i + 1
		if argIdx < len(site.argumentStaticTypeStrings) {
			raw = restoreNamedTypeForFmt(vm, raw, site.argumentStaticTypeStrings[argIdx])
		}
		varArgs[i] = wrapPikoSynthesisedFmtArg(vm, raw)
	}

	if rewrittenFormat, rewrittenArgs, intercepted := interceptFmtFormat(site, 1, format, varArgs); intercepted {
		format = rewrittenFormat
		varArgs = rewrittenArgs
	}

	result := f(format, varArgs...)
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchSprintVarargs dispatches a native function with the signature func(...any)
// string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintVarargs(vm *VM, cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(...any) string)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}

	varArgs := site.reuseVarArgsBuf(len(site.arguments))
	for i := range site.arguments {
		raw := readAnyArg(registers, site.arguments[i])
		if i < len(site.argumentStaticTypeStrings) {
			raw = restoreNamedTypeForFmt(vm, raw, site.argumentStaticTypeStrings[i])
		}
		varArgs[i] = wrapPikoSynthesisedFmtArg(vm, raw)
	}

	registers.strings[site.returns[0].register] = f(varArgs...)
}

// fpDispatchBytesArrN dispatches a digest-returning native call.
//
// Generic body shared by fpDispatchBytesArr16/20/32/64. The fixed-size array type A is
// the digest type the cached function returns; allocSize is its byte width; abiType is
// the pre-computed *abi.Type token for the result reflect.Value. Specialising on A lets
// each instantiation become a concrete monomorphised function with no runtime dispatch -
// the type assertion `cached.(func([]byte) A)` and the typed store `*(*A)(slot) = result`
// resolve at compile time per A.
//
// Takes vm (*VM) which provides the arena allocator.
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
// Takes abiType (unsafe.Pointer) which is the result type's *abi.Type token (one of
// bytesArr16/20/32/64ABIType).
// Takes allocSize (uintptr) which is sizeof(A) for the arena slab reservation.
func fpDispatchBytesArrN[A any](
	vm *VM, cached any, site *callSite, registers *Registers,
	abiType unsafe.Pointer, allocSize uintptr,
) {
	f, ok := cached.(func([]byte) A)
	if !ok {
		_ = recordFastPathTypeMismatch(vm, -1, fastPathDispatcherTag, reflectTypeName(cached))
		return
	}
	result := f(readBytesArg(registers, site.arguments[0]))
	if vm.arena == nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
		return
	}
	slot := vm.arena.AllocBytes(allocSize, 1)
	*(*A)(slot) = result
	registers.general[site.returns[0].register] = unsafeNewAt(abiType, slot, reflect.Array)
}

// fpDispatchBytesArr32 dispatches func([]byte) [32]byte.
//
// SHA-256 over thousands of lines is the canonical workload.
//
// Takes vm (*VM) which provides the arena allocator.
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchBytesArr32(vm *VM, cached any, site *callSite, registers *Registers) {
	fpDispatchBytesArrN[[digestBytesSHA256]byte](vm, cached, site, registers, bytesArr32ABIType, digestBytesSHA256)
}

// fpDispatchBytesArr16 dispatches func([]byte) [16]byte.
//
// MD5 / UUID digest sizes.
//
// Takes vm (*VM) which provides the arena allocator.
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchBytesArr16(vm *VM, cached any, site *callSite, registers *Registers) {
	fpDispatchBytesArrN[[digestBytesMD5]byte](vm, cached, site, registers, bytesArr16ABIType, digestBytesMD5)
}

// fpDispatchBytesArr20 dispatches func([]byte) [20]byte.
//
// SHA-1 digest size.
//
// Takes vm (*VM) which provides the arena allocator.
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchBytesArr20(vm *VM, cached any, site *callSite, registers *Registers) {
	fpDispatchBytesArrN[[digestBytesSHA1]byte](vm, cached, site, registers, bytesArr20ABIType, digestBytesSHA1)
}

// fpDispatchBytesArr64 dispatches func([]byte) [64]byte.
//
// SHA-512 digest size.
//
// Takes vm (*VM) which provides the arena allocator.
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchBytesArr64(vm *VM, cached any, site *callSite, registers *Registers) {
	fpDispatchBytesArrN[[digestBytesSHA512]byte](vm, cached, site, registers, bytesArr64ABIType, digestBytesSHA512)
}

// readBytesArg reads a []byte argument from the appropriate register bank. Mirrors
// readStringArg / readIntArg shape.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the []byte from the slicesByte bank when kind is registerSliceByte; otherwise
// reads via .Bytes() on the general bank reflect.Value.
func readBytesArg(registers *Registers, argument varLocation) []byte {
	if argument.kind == registerSliceByte {
		return registers.slicesByte[argument.register]
	}
	v := registers.general[argument.register]
	if v.IsValid() && v.Kind() == reflect.Slice {
		return v.Bytes()
	}
	return nil
}

// readStringArg reads a string argument from the appropriate register bank based on the
// argument's kind.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the string value from the string register bank if the argument kind is
// registerString, or converts from the general register bank otherwise.
func readStringArg(registers *Registers, argument varLocation) string {
	if argument.kind == registerString {
		return registers.strings[argument.register]
	}

	return registers.general[argument.register].String()
}

// readIntArg reads an int argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the int64 value from the int register bank if the argument kind is registerInt,
// or converts from the general register bank otherwise.
func readIntArg(registers *Registers, argument varLocation) int64 {
	if argument.kind == registerInt {
		return registers.ints[argument.register]
	}

	return registers.general[argument.register].Int()
}

// readFloatArg reads a float argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the float64 value from the float register bank if the argument kind is
// registerFloat, or converts from the general register bank otherwise.
func readFloatArg(registers *Registers, argument varLocation) float64 {
	if argument.kind == registerFloat {
		return registers.floats[argument.register]
	}

	return registers.general[argument.register].Float()
}

// readBoolArg reads a bool argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the bool value from the bool register bank if the argument kind is
// registerBool, or converts from the general register bank otherwise.
func readBoolArg(registers *Registers, argument varLocation) bool {
	if argument.kind == registerBool {
		return registers.bools[argument.register]
	}

	return registers.general[argument.register].Bool()
}

// readAnyArg reads an argument from any register bank and returns it as an interface{}
// value. Used for variadic fast paths where arguments must be boxed into []any.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index to read.
//
// Returns the value from the appropriate typed register bank based on the argument kind,
// or nil if the kind is unrecognised or the value is invalid.
func readAnyArg(registers *Registers, argument varLocation) any {
	switch argument.kind {
	case registerInt:
		return registers.ints[argument.register]
	case registerFloat:
		return registers.floats[argument.register]
	case registerString:
		return registers.strings[argument.register]
	case registerBool:
		return registers.bools[argument.register]
	case registerUint:
		return registers.uints[argument.register]
	case registerComplex:
		return registers.complex[argument.register]
	case registerGeneral:
		v := registers.general[argument.register]
		if v.IsValid() {
			return v.Interface()
		}

		return nil
	default:
		return nil
	}
}

func init() {
	fastPathDispatchTable[fastPathTagStringString] = fpDispatchStringString
	fastPathDispatchTable[fastPathTagStringInt] = fpDispatchStringInt
	fastPathDispatchTable[fastPathTagStringBool] = fpDispatchStringBool
	fastPathDispatchTable[fastPathTagStringRuneBool] = fpDispatchStringRuneBool
	fastPathDispatchTable[fastPathTagStringRuneInt] = fpDispatchStringRuneInt
	fastPathDispatchTable[fastPathTagString2Bool] = fpDispatchString2Bool
	fastPathDispatchTable[fastPathTagString2String] = fpDispatchString2String
	fastPathDispatchTable[fastPathTagString2Int] = fpDispatchString2Int
	fastPathDispatchTable[fastPathTagString3String] = fpDispatchString3String
	fastPathDispatchTable[fastPathTagIntString] = fpDispatchIntString
	fastPathDispatchTable[fastPathTagIntInt] = fpDispatchIntInt
	fastPathDispatchTable[fastPathTagIntBool] = fpDispatchIntBool
	fastPathDispatchTable[fastPathTagInt2Int] = fpDispatchInt2Int
	fastPathDispatchTable[fastPathTagInt2Bool] = fpDispatchInt2Bool
	fastPathDispatchTable[fastPathTagInt2String] = fpDispatchInt2String
	fastPathDispatchTable[fastPathTagInt64IntString] = fpDispatchInt64IntString
	fastPathDispatchTable[fastPathTagStringIntError] = fpDispatchStringIntError
	fastPathDispatchTable[fastPathTagFloat64Float64] = fpDispatchFloat64Float64
	fastPathDispatchTable[fastPathTagFloat642Float64] = fpDispatchFloat642Float64
	fastPathDispatchTable[fastPathTagAnyBool] = fpDispatchAnyBool
	fastPathDispatchTable[fastPathTagAnyString] = fpDispatchAnyString
	fastPathDispatchTable[fastPathTagAnyInt] = fpDispatchAnyInt
	fastPathDispatchTable[fastPathTagAnyInt64] = fpDispatchAnyInt64
	fastPathDispatchTable[fastPathTagAnyFloat64] = fpDispatchAnyFloat64
	fastPathDispatchTable[fastPathTagAny2Any] = fpDispatchAny2Any
	fastPathDispatchTable[fastPathTagRetString] = fpDispatchRetString
	fastPathDispatchTable[fastPathTagRetBool] = fpDispatchRetBool
	fastPathDispatchTable[fastPathTagRetInt] = fpDispatchRetInt
	fastPathDispatchTable[fastPathTagRetInt64] = fpDispatchRetInt64
	fastPathDispatchTable[fastPathTagRetFloat64] = fpDispatchRetFloat64
	fastPathDispatchTable[fastPathTagRetError] = fpDispatchRetError
	fastPathDispatchTable[fastPathTagVoid] = fpDispatchVoid
	fastPathDispatchTable[fastPathTagVoidString] = fpDispatchVoidString
	fastPathDispatchTable[fastPathTagVoidInt] = fpDispatchVoidInt
	fastPathDispatchTable[fastPathTagVoidInt64] = fpDispatchVoidInt64
	fastPathDispatchTable[fastPathTagVoidBool] = fpDispatchVoidBool
	fastPathDispatchTable[fastPathTagVoidString2] = fpDispatchVoidString2
	fastPathDispatchTable[fastPathTagStringError] = fpDispatchStringError
	fastPathDispatchTable[fastPathTagSprintfString] = fpDispatchSprintfString
	fastPathDispatchTable[fastPathTagSprintfError] = fpDispatchSprintfError
	fastPathDispatchTable[fastPathTagSprintVarargs] = fpDispatchSprintVarargs
	fastPathDispatchTable[fastPathTagBytesArr32] = fpDispatchBytesArr32
	fastPathDispatchTable[fastPathTagBytesArr16] = fpDispatchBytesArr16
	fastPathDispatchTable[fastPathTagBytesArr20] = fpDispatchBytesArr20
	fastPathDispatchTable[fastPathTagBytesArr64] = fpDispatchBytesArr64
}
