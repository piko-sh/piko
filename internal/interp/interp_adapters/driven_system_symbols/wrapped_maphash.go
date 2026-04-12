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

package driven_system_symbols

import (
	"fmt"
	"hash/maphash"
	"reflect"
)

// wrappedMaphashComparable bridges hash/maphash.Comparable for interpreted code.
//
// Computes a 64-bit hash of value derived from seed. Uses a type switch over built-in
// comparable kinds; falls back to hashing the value's reflect.Interface() - equivalent to
// wrapping value in any and calling Comparable[any] - for everything else.
//
// Takes seed (maphash.Seed) which provides the hash key.
// Takes value (any) which must be a comparable Go value.
//
// Returns uint64 which is the computed hash.
//
// Panics with "hash/maphash.Comparable: incomparable type %T" when value is not
// comparable (slice, map, function).
func wrappedMaphashComparable(seed maphash.Seed, value any) uint64 {
	switch typed := value.(type) {
	case string:
		return maphash.Comparable(seed, typed)
	case bool:
		return maphash.Comparable(seed, typed)
	case int:
		return maphash.Comparable(seed, typed)
	case int8:
		return maphash.Comparable(seed, typed)
	case int16:
		return maphash.Comparable(seed, typed)
	case int32:
		return maphash.Comparable(seed, typed)
	case int64:
		return maphash.Comparable(seed, typed)
	case uint:
		return maphash.Comparable(seed, typed)
	case uint8:
		return maphash.Comparable(seed, typed)
	case uint16:
		return maphash.Comparable(seed, typed)
	case uint32:
		return maphash.Comparable(seed, typed)
	case uint64:
		return maphash.Comparable(seed, typed)
	case uintptr:
		return maphash.Comparable(seed, typed)
	case float32:
		return maphash.Comparable(seed, typed)
	case float64:
		return maphash.Comparable(seed, typed)
	case complex64:
		return maphash.Comparable(seed, typed)
	case complex128:
		return maphash.Comparable(seed, typed)
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return maphash.Comparable(seed, any(nil))
	}
	if !rv.Comparable() {
		panic(fmt.Sprintf("hash/maphash.Comparable: incomparable type %T", value))
	}
	return maphash.Comparable(seed, value)
}

// wrappedMaphashWriteComparable bridges hash/maphash.WriteComparable[T]. Same dispatch
// strategy as wrappedMaphashComparable but writes into an existing maphash.Hash rather
// than returning a one-shot result.
//
// Takes hash (*maphash.Hash) which is the hash state to mutate.
// Takes value (any) which must be a comparable Go value.
//
// Panics with "hash/maphash.WriteComparable: incomparable type %T" when value is not
// comparable.
func wrappedMaphashWriteComparable(hash *maphash.Hash, value any) {
	if writeMaphashBuiltin(hash, value) {
		return
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		maphash.WriteComparable(hash, any(nil))
		return
	}
	if !rv.Comparable() {
		panic(fmt.Sprintf("hash/maphash.WriteComparable: incomparable type %T", value))
	}
	maphash.WriteComparable(hash, value)
}

// writeMaphashBuiltin writes value into hash via maphash.WriteComparable when value is
// one of the built-in comparable kinds, dispatching on the concrete runtime type so the
// per-T hashing contract is preserved.
//
// Takes hash (*maphash.Hash) the hash state to mutate and value (any) the candidate
// value.
//
// Returns true when value matched a built-in kind and was written; false when the caller
// must use the reflective fallback.
func writeMaphashBuiltin(hash *maphash.Hash, value any) bool {
	switch typed := value.(type) {
	case string:
		maphash.WriteComparable(hash, typed)
	case bool:
		maphash.WriteComparable(hash, typed)
	case int:
		maphash.WriteComparable(hash, typed)
	case int8:
		maphash.WriteComparable(hash, typed)
	case int16:
		maphash.WriteComparable(hash, typed)
	case int32:
		maphash.WriteComparable(hash, typed)
	case int64:
		maphash.WriteComparable(hash, typed)
	case uint:
		maphash.WriteComparable(hash, typed)
	case uint8:
		maphash.WriteComparable(hash, typed)
	case uint16:
		maphash.WriteComparable(hash, typed)
	case uint32:
		maphash.WriteComparable(hash, typed)
	case uint64:
		maphash.WriteComparable(hash, typed)
	case uintptr:
		maphash.WriteComparable(hash, typed)
	case float32:
		maphash.WriteComparable(hash, typed)
	case float64:
		maphash.WriteComparable(hash, typed)
	case complex64:
		maphash.WriteComparable(hash, typed)
	case complex128:
		maphash.WriteComparable(hash, typed)
	default:
		return false
	}
	return true
}

func init() {
	if _, ok := Symbols["hash/maphash"]; ok {
		Symbols["hash/maphash"]["Comparable"] = reflect.ValueOf(wrappedMaphashComparable)
		Symbols["hash/maphash"]["WriteComparable"] = reflect.ValueOf(wrappedMaphashWriteComparable)
	}
}
