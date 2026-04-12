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
	"sync"
)

const (
	// numRegisterKindEntries sizes kindZeroValueCache to cover the registerKind enum plus a
	// fallback "any" slot at the end.
	numRegisterKindEntries = NumRegisterKinds + 1
)

var (
	// kindZeroValueCache holds pre-computed reflect.Zero values per register kind.
	//
	// Populated in init() from the same types kindDefaultReflectType returns; lookup is a
	// single slice index (~1 ns) vs the ~50 ns allocation reflect.Zero incurs. Each zero
	// value is immutable; sharing the same reflect.Value across all callers is safe.
	kindZeroValueCache [numRegisterKindEntries]reflect.Value

	// typeZeroValueCache deduplicates reflect.Zero(t) calls.
	//
	// Caches the immutable result per reflect.Type. Zero values are content-immutable;
	// sharing one instance across callers is correct for every consumer that reads but does
	// not mutate (which is every caller of reflect.Zero - the value is conceptually a
	// constant zero of type t).
	typeZeroValueCache sync.Map
)

func init() {
	kindZeroValueCache[registerInt] = reflect.Zero(reflect.TypeFor[int64]())
	kindZeroValueCache[registerFloat] = reflect.Zero(reflect.TypeFor[float64]())
	kindZeroValueCache[registerString] = reflect.Zero(reflect.TypeFor[string]())
	kindZeroValueCache[registerBool] = reflect.Zero(reflect.TypeFor[bool]())
	kindZeroValueCache[registerUint] = reflect.Zero(reflect.TypeFor[uint64]())
	kindZeroValueCache[registerComplex] = reflect.Zero(reflect.TypeFor[complex128]())
	kindZeroValueCache[NumRegisterKinds] = reflect.Zero(reflect.TypeFor[any]())
}

// zeroValueForKind returns the cached zero value for a register kind.
//
// Replaces the `reflect.Zero(kindDefaultReflectType(k))` pattern at every call site. Each
// call would otherwise mallocgc a fresh reflect.Value (~50 ns + 24 B); the cache returns
// the same shared zero value (~1 ns, 0 B).
//
// Takes k (registerKind) which selects the kind.
//
// Returns the immutable cached zero reflect.Value for that kind. For kinds outside the
// scalar set (typed-slice banks, etc.) returns the `any` zero - matching
// kindDefaultReflectType's fallback.
func zeroValueForKind(k registerKind) reflect.Value {
	switch k {
	case registerInt, registerFloat, registerString, registerBool, registerUint, registerComplex:
		return kindZeroValueCache[k]
	default:
	}
	return kindZeroValueCache[NumRegisterKinds]
}

// zeroValueForType returns the cached reflect.Zero for type t.
//
// On cache miss, allocates via reflect.Zero and publishes. Used by hot-path handlers
// (handleTypeAssert no-match, map index miss, channel recv default) where reflect.Zero
// would otherwise allocate per call.
//
// Takes t (reflect.Type) which is the type whose zero value is requested. May be nil; nil
// returns the invalid reflect.Value.
//
// Returns the cached immutable zero value for t.
func zeroValueForType(t reflect.Type) reflect.Value {
	if t == nil {
		return reflect.Value{}
	}
	if cached, ok := typeZeroValueCache.Load(t); ok {
		if cachedValue, isValue := cached.(reflect.Value); isValue {
			return cachedValue
		}
	}
	zero := reflect.Zero(t)
	typeZeroValueCache.Store(t, zero)
	return zero
}
