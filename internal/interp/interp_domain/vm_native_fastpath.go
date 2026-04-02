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

	"piko.sh/piko/wdk/safeconv"
)

// nativeFastPathTag identifies which fast-path case matched so that
// subsequent calls can dispatch via a uint8 jump table instead of
// the full interface type switch.
type nativeFastPathTag uint8

const (
	// fastPathTagNone indicates that no fast-path dispatch is available for this
	// call site after classification has been attempted. The zero value
	// (unprobed) is implicit - any call site starts at 0 before classification.
	fastPathTagNone nativeFastPathTag = iota + 1

	// fastPathTagStringString represents the fast-path tag for native functions
	// with the signature func(string) string.
	fastPathTagStringString

	// fastPathTagStringInt represents the fast-path tag for native functions
	// with the signature func(string) int.
	fastPathTagStringInt

	// fastPathTagStringBool represents the fast-path tag for native functions
	// with the signature func(string) bool.
	fastPathTagStringBool

	// fastPathTagStringRuneBool represents the fast-path tag for native functions
	// with the signature func(string, int32) bool.
	fastPathTagStringRuneBool

	// fastPathTagStringRuneInt represents the fast-path tag for native functions
	// with the signature func(string, int32) int.
	fastPathTagStringRuneInt

	// fastPathTagString2Bool represents the fast-path tag for native functions
	// with the signature func(string, string) bool.
	fastPathTagString2Bool

	// fastPathTagString2String represents the fast-path tag for native functions
	// with the signature func(string, string) string.
	fastPathTagString2String

	// fastPathTagString2Int represents the fast-path tag for native functions
	// with the signature func(string, string) int.
	fastPathTagString2Int

	// fastPathTagString3String represents the fast-path tag for native functions
	// with the signature func(string, string, string) string.
	fastPathTagString3String

	// fastPathTagIntString represents the fast-path tag for native functions
	// with the signature func(int) string.
	fastPathTagIntString

	// fastPathTagIntInt represents the fast-path tag for native functions
	// with the signature func(int) int.
	fastPathTagIntInt

	// fastPathTagIntBool represents the fast-path tag for native functions
	// with the signature func(int) bool.
	fastPathTagIntBool

	// fastPathTagInt2Int represents the fast-path tag for native functions
	// with the signature func(int, int) int.
	fastPathTagInt2Int

	// fastPathTagInt2Bool represents the fast-path tag for native functions
	// with the signature func(int, int) bool.
	fastPathTagInt2Bool

	// fastPathTagInt2String represents the fast-path tag for native functions
	// with the signature func(int, int) string.
	fastPathTagInt2String

	// fastPathTagInt64IntString represents the fast-path tag for native functions
	// with the signature func(int64, int) string.
	fastPathTagInt64IntString

	// fastPathTagStringIntError represents the fast-path tag for native functions
	// with the signature func(string) (int, error).
	fastPathTagStringIntError

	// fastPathTagFloat64Float64 represents the fast-path tag for native functions
	// with the signature func(float64) float64.
	fastPathTagFloat64Float64

	// fastPathTagFloat642Float64 represents the fast-path tag for native functions
	// with the signature func(float64, float64) float64.
	fastPathTagFloat642Float64

	// fastPathTagAnyBool represents the fast-path tag for native functions
	// with the signature func(any) bool.
	fastPathTagAnyBool

	// fastPathTagAnyString represents the fast-path tag for native functions
	// with the signature func(any) string.
	fastPathTagAnyString

	// fastPathTagAnyInt represents the fast-path tag for native functions
	// with the signature func(any) int.
	fastPathTagAnyInt

	// fastPathTagAnyInt64 represents the fast-path tag for native functions
	// with the signature func(any) int64.
	fastPathTagAnyInt64

	// fastPathTagAnyFloat64 represents the fast-path tag for native functions
	// with the signature func(any) float64.
	fastPathTagAnyFloat64

	// fastPathTagAny2Any represents the fast-path tag for native functions
	// with the signature func(any, any) any.
	fastPathTagAny2Any

	// fastPathTagRetString represents the fast-path tag for native functions
	// with the signature func() string.
	fastPathTagRetString

	// fastPathTagRetBool represents the fast-path tag for native functions
	// with the signature func() bool.
	fastPathTagRetBool

	// fastPathTagRetInt represents the fast-path tag for native functions
	// with the signature func() int.
	fastPathTagRetInt

	// fastPathTagRetInt64 represents the fast-path tag for native functions
	// with the signature func() int64.
	fastPathTagRetInt64

	// fastPathTagRetFloat64 represents the fast-path tag for native functions
	// with the signature func() float64.
	fastPathTagRetFloat64

	// fastPathTagRetError represents the fast-path tag for native functions
	// with the signature func() error.
	fastPathTagRetError

	// fastPathTagVoid represents the fast-path tag for native functions
	// with the signature func().
	fastPathTagVoid

	// fastPathTagVoidString represents the fast-path tag for native functions
	// with the signature func(string).
	fastPathTagVoidString

	// fastPathTagVoidInt represents the fast-path tag for native functions
	// with the signature func(int).
	fastPathTagVoidInt

	// fastPathTagVoidInt64 represents the fast-path tag for native functions
	// with the signature func(int64).
	fastPathTagVoidInt64

	// fastPathTagVoidBool represents the fast-path tag for native functions
	// with the signature func(bool).
	fastPathTagVoidBool

	// fastPathTagVoidString2 represents the fast-path tag for native functions
	// with the signature func(string, string).
	fastPathTagVoidString2

	// fastPathTagStringError represents the fast-path tag for native functions
	// with the signature func(string) error.
	fastPathTagStringError

	// fastPathTagSprintfString represents the fast-path tag for native functions
	// with the signature func(string, ...any) string.
	fastPathTagSprintfString

	// fastPathTagSprintfError represents the fast-path tag for native functions
	// with the signature func(string, ...any) error.
	fastPathTagSprintfError

	// fastPathTagSprintVarargs represents the fast-path tag for native functions
	// with the signature func(...any) string.
	fastPathTagSprintVarargs
)

var (
	// nativeFastPathNone is the sentinel value indicating that a call site
	// has no fast-path specialisation available.
	nativeFastPathNone = &struct{}{}

	// fastPathTagByType maps concrete function reflect.Types to their
	// corresponding fast-path tag for O(1) classification.
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
	}

	// fastPathDispatchTable is an array of dispatch functions indexed by
	// nativeFastPathTag. Populated at init time.
	fastPathDispatchTable [fastPathTagSprintVarargs + 1]fastPathDispatcher
)

// fastPathDispatcher is the signature for individual fast-path
// dispatch functions, each handling exactly one tag case.
type fastPathDispatcher func(cached any, site *callSite, registers *Registers)

// reuseVarArgsBuf returns a []any slice of length n, reusing the
// pre-allocated buffer on the callSite when possible.
//
// Takes n (int) which is the required slice length.
//
// Returns a []any slice of the requested length, either resliced from
// the existing buffer or freshly allocated if the buffer capacity is
// insufficient.
func (cs *callSite) reuseVarArgsBuf(n int) []any {
	if cap(cs.variadicArgumentsBuffer) >= n {
		return cs.variadicArgumentsBuffer[:n]
	}

	cs.variadicArgumentsBuffer = make([]any, n)

	return cs.variadicArgumentsBuffer
}

// classifyNativeFastPath determines the fast-path tag for a given
// function value by looking up its reflect.Type in the dispatch map.
//
// Takes v (any) which is the native function value to classify.
//
// Returns the matching nativeFastPathTag, or fastPathTagNone if no
// fast-path specialisation exists for the function's type signature.
func classifyNativeFastPath(v any) nativeFastPathTag {
	tag, ok := fastPathTagByType[reflect.TypeOf(v)]
	if !ok {
		return fastPathTagNone
	}

	return tag
}

// tryNativeFastPath attempts to call a native function via a direct
// type assertion on the already-extracted function value, bypassing
// reflect.Value.Call().
//
// The caller is responsible for extracting the function value (via
// reflectedFunction.Interface()) and passing it as v. For handleCallNative, the
// extracted value is cached on the callSite for subsequent calls,
// eliminating repeated Interface() allocations. For handleCall
// (closure dispatch), the value is extracted fresh each time because
// the underlying function may change between calls.
//
// Takes site (*callSite) which is the call site metadata including
// argument and return register locations.
// Takes v (any) which is the native function value to dispatch.
// Takes registers (*Registers) which is the VM register file to read
// arguments from and write results to.
//
// Returns true and the matched tag if the fast path was taken, or false
// and fastPathTagNone if no fast-path specialisation is available.
func tryNativeFastPath(site *callSite, v any, registers *Registers) (bool, nativeFastPathTag) {
	tag := classifyNativeFastPath(v)
	if tag == fastPathTagNone {
		site.nativeFastPath = nativeFastPathNone
		return false, fastPathTagNone
	}

	dispatchNativeFastPathTagged(tag, v, site, registers)

	return true, tag
}

// dispatchNativeFastPathTagged dispatches a cached native call using
// the pre-resolved tag via an array lookup, avoiding the sequential
// itab comparisons of a full type switch.
//
// Takes tag (nativeFastPathTag) which is the pre-classified fast-path tag.
// Takes cached (any) which is the native function value extracted via
// Interface().
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file to read
// arguments from and write results to.
func dispatchNativeFastPathTagged(tag nativeFastPathTag, cached any, site *callSite, registers *Registers) {
	if int(tag) < len(fastPathDispatchTable) {
		if d := fastPathDispatchTable[tag]; d != nil {
			d(cached, site, registers)
		}
	}
}

// fpDispatchStringString dispatches a native function with the signature
// func(string) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchStringInt dispatches a native function with the signature
// func(string) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(readStringArg(registers, site.arguments[0])))
}

// fpDispatchStringBool dispatches a native function with the signature
// func(string) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchStringRuneBool dispatches a native function with the signature
// func(string, int32) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringRuneBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, int32) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		safeconv.Int64ToInt32(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchStringRuneInt dispatches a native function with the signature
// func(string, int32) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringRuneInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, int32) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		readStringArg(registers, site.arguments[0]),
		safeconv.Int64ToInt32(readIntArg(registers, site.arguments[1])),
	))
}

// fpDispatchString2Bool dispatches a native function with the signature
// func(string, string) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2Bool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	)
}

// fpDispatchString2String dispatches a native function with the signature
// func(string, string) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2String(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	)
}

// fpDispatchString2Int dispatches a native function with the signature
// func(string, string) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString2Int(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
	))
}

// fpDispatchString3String dispatches a native function with the signature
// func(string, string, string) string via direct type assertion on the
// cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchString3String(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string, string) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(
		readStringArg(registers, site.arguments[0]),
		readStringArg(registers, site.arguments[1]),
		readStringArg(registers, site.arguments[2]),
	)
}

// fpDispatchIntString dispatches a native function with the signature
// func(int) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchIntInt dispatches a native function with the signature
// func(int) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(int(readIntArg(registers, site.arguments[0]))))
}

// fpDispatchIntBool dispatches a native function with the signature
// func(int) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchIntBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchInt2Int dispatches a native function with the signature
// func(int, int) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2Int(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	))
}

// fpDispatchInt2Bool dispatches a native function with the signature
// func(int, int) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2Bool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchInt2String dispatches a native function with the signature
// func(int, int) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt2String(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int, int) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(
		int(readIntArg(registers, site.arguments[0])),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchInt64IntString dispatches a native function with the signature
// func(int64, int) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchInt64IntString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int64, int) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(
		readIntArg(registers, site.arguments[0]),
		int(readIntArg(registers, site.arguments[1])),
	)
}

// fpDispatchStringIntError dispatches a native function with the signature
// func(string) (int, error) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringIntError(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) (int, error))
	if !ok {
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

// fpDispatchFloat64Float64 dispatches a native function with the signature
// func(float64) float64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchFloat64Float64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(float64) float64)
	if !ok {
		return
	}

	registers.floats[site.returns[0].register] = f(readFloatArg(registers, site.arguments[0]))
}

// fpDispatchFloat642Float64 dispatches a native function with the signature
// func(float64, float64) float64 via direct type assertion on the cached
// value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchFloat642Float64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(float64, float64) float64)
	if !ok {
		return
	}

	registers.floats[site.returns[0].register] = f(
		readFloatArg(registers, site.arguments[0]),
		readFloatArg(registers, site.arguments[1]),
	)
}

// fpDispatchAnyBool dispatches a native function with the signature
// func(any) bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f(readAnyArg(registers, site.arguments[0]))
}

// fpDispatchAnyString dispatches a native function with the signature
// func(any) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f(readAnyArg(registers, site.arguments[0]))
}

// fpDispatchAnyInt dispatches a native function with the signature
// func(any) int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f(readAnyArg(registers, site.arguments[0])))
}

// fpDispatchAnyInt64 dispatches a native function with the signature
// func(any) int64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyInt64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) int64)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = f(readAnyArg(registers, site.arguments[0]))
}

// fpDispatchAnyFloat64 dispatches a native function with the signature
// func(any) float64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAnyFloat64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any) float64)
	if !ok {
		return
	}

	registers.floats[site.returns[0].register] = f(readAnyArg(registers, site.arguments[0]))
}

// fpDispatchAny2Any dispatches a native function with the signature
// func(any, any) any via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchAny2Any(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(any, any) any)
	if !ok {
		return
	}

	result := f(readAnyArg(registers, site.arguments[0]), readAnyArg(registers, site.arguments[1]))
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchRetString dispatches a native function with the signature
// func() string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() string)
	if !ok {
		return
	}

	registers.strings[site.returns[0].register] = f()
}

// fpDispatchRetBool dispatches a native function with the signature
// func() bool via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() bool)
	if !ok {
		return
	}

	registers.bools[site.returns[0].register] = f()
}

// fpDispatchRetInt dispatches a native function with the signature
// func() int via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() int)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = int64(f())
}

// fpDispatchRetInt64 dispatches a native function with the signature
// func() int64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetInt64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() int64)
	if !ok {
		return
	}

	registers.ints[site.returns[0].register] = f()
}

// fpDispatchRetFloat64 dispatches a native function with the signature
// func() float64 via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetFloat64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() float64)
	if !ok {
		return
	}

	registers.floats[site.returns[0].register] = f()
}

// fpDispatchRetError dispatches a native function with the signature
// func() error via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchRetError(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func() error)
	if !ok {
		return
	}

	result := f()
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchVoid dispatches a native function with the signature
// func() via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
func fpDispatchVoid(cached any, _ *callSite, _ *Registers) {
	f, ok := cached.(func())
	if !ok {
		return
	}

	f()
}

// fpDispatchVoidString dispatches a native function with the signature
// func(string) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string))
	if !ok {
		return
	}

	f(readStringArg(registers, site.arguments[0]))
}

// fpDispatchVoidInt dispatches a native function with the signature
// func(int) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidInt(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int))
	if !ok {
		return
	}

	f(int(readIntArg(registers, site.arguments[0])))
}

// fpDispatchVoidInt64 dispatches a native function with the signature
// func(int64) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidInt64(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(int64))
	if !ok {
		return
	}

	f(readIntArg(registers, site.arguments[0]))
}

// fpDispatchVoidBool dispatches a native function with the signature
// func(bool) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidBool(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(bool))
	if !ok {
		return
	}

	f(readBoolArg(registers, site.arguments[0]))
}

// fpDispatchVoidString2 dispatches a native function with the signature
// func(string, string) via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchVoidString2(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, string))
	if !ok {
		return
	}

	f(readStringArg(registers, site.arguments[0]), readStringArg(registers, site.arguments[1]))
}

// fpDispatchStringError dispatches a native function with the signature
// func(string) error via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchStringError(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string) error)
	if !ok {
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

// fpDispatchSprintfString dispatches a native function with the signature
// func(string, ...any) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintfString(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, ...any) string)
	if !ok {
		return
	}

	format := readStringArg(registers, site.arguments[0])
	nVarArgs := len(site.arguments) - 1
	varArgs := site.reuseVarArgsBuf(nVarArgs)

	for i := range nVarArgs {
		varArgs[i] = readAnyArg(registers, site.arguments[i+1])
	}

	registers.strings[site.returns[0].register] = f(format, varArgs...)
}

// fpDispatchSprintfError dispatches a native function with the signature
// func(string, ...any) error via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintfError(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(string, ...any) error)
	if !ok {
		return
	}

	format := readStringArg(registers, site.arguments[0])
	nVarArgs := len(site.arguments) - 1
	varArgs := site.reuseVarArgsBuf(nVarArgs)

	for i := range nVarArgs {
		varArgs[i] = readAnyArg(registers, site.arguments[i+1])
	}

	result := f(format, varArgs...)
	if result != nil {
		registers.general[site.returns[0].register] = reflect.ValueOf(result)
	} else {
		registers.general[site.returns[0].register] = reflect.Value{}
	}
}

// fpDispatchSprintVarargs dispatches a native function with the signature
// func(...any) string via direct type assertion on the cached value.
//
// Takes cached (any) which is the native function value to assert.
// Takes site (*callSite) which is the call site metadata.
// Takes registers (*Registers) which is the VM register file.
func fpDispatchSprintVarargs(cached any, site *callSite, registers *Registers) {
	f, ok := cached.(func(...any) string)
	if !ok {
		return
	}

	varArgs := site.reuseVarArgsBuf(len(site.arguments))
	for i := range site.arguments {
		varArgs[i] = readAnyArg(registers, site.arguments[i])
	}

	registers.strings[site.returns[0].register] = f(varArgs...)
}

// readStringArg reads a string argument from the appropriate register
// bank based on the argument's kind.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index
// to read.
//
// Returns the string value from the string register bank if the argument
// kind is registerString, or converts from the general register bank otherwise.
func readStringArg(registers *Registers, argument varLocation) string {
	if argument.kind == registerString {
		return registers.strings[argument.register]
	}

	return registers.general[argument.register].String()
}

// readIntArg reads an int argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index
// to read.
//
// Returns the int64 value from the int register bank if the argument
// kind is registerInt, or converts from the general register bank otherwise.
func readIntArg(registers *Registers, argument varLocation) int64 {
	if argument.kind == registerInt {
		return registers.ints[argument.register]
	}

	return registers.general[argument.register].Int()
}

// readFloatArg reads a float argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index
// to read.
//
// Returns the float64 value from the float register bank if the argument
// kind is registerFloat, or converts from the general register bank otherwise.
func readFloatArg(registers *Registers, argument varLocation) float64 {
	if argument.kind == registerFloat {
		return registers.floats[argument.register]
	}

	return registers.general[argument.register].Float()
}

// readBoolArg reads a bool argument from the appropriate register bank.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index
// to read.
//
// Returns the bool value from the bool register bank if the argument
// kind is registerBool, or converts from the general register bank otherwise.
func readBoolArg(registers *Registers, argument varLocation) bool {
	if argument.kind == registerBool {
		return registers.bools[argument.register]
	}

	return registers.general[argument.register].Bool()
}

// readAnyArg reads an argument from any register bank and returns it
// as an interface{} value. Used for variadic fast paths where arguments
// must be boxed into []any.
//
// Takes registers (*Registers) which is the VM register file to read from.
// Takes argument (varLocation) which describes the register bank and index
// to read.
//
// Returns the value from the appropriate typed register bank based on the
// argument kind, or nil if the kind is unrecognised or the value is invalid.
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
}
