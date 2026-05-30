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
	"reflect"
)

// Reflective fallbacks for closure-taking generic stdlib functions where the caller
// supplies a func(E[, E|T]) bool or func(E, E) int predicate. The generated wrappers pass
// these closures with `any` element types (e.g. func(any, any) int) so each helper can
// invoke the closure directly without further reflection.

// reflectSlicesContainsFunc reports whether any element of s satisfies f. Equivalent to
// slices.ContainsFunc.
//
// Takes s (any) which must be a slice, and f (func(any) bool) the predicate.
//
// Returns true if any element satisfies f.
func reflectSlicesContainsFunc(s any, f func(any) bool) bool {
	rv := requireSlice("slices.ContainsFunc", s)
	for i := range rv.Len() {
		if f(rv.Index(i).Interface()) {
			return true
		}
	}
	return false
}

// reflectSlicesIndexFunc returns the first index for which f returns true, or -1 when no
// element matches. Equivalent to slices.IndexFunc.
//
// Takes s (any) which must be a slice, and f (func(any) bool).
//
// Returns the index of the first match, or -1.
func reflectSlicesIndexFunc(s any, f func(any) bool) int {
	rv := requireSlice("slices.IndexFunc", s)
	for i := range rv.Len() {
		if f(rv.Index(i).Interface()) {
			return i
		}
	}
	return -1
}

// reflectSlicesIsSortedFunc reports whether x is sorted under cmp. Equivalent to
// slices.IsSortedFunc.
//
// Takes x (any) which must be a slice, and cmp (func(any, any) int) returning negative if
// a < b, zero if equal, positive if a > b.
//
// Returns true if x is sorted.
func reflectSlicesIsSortedFunc(x any, cmp func(any, any) int) bool {
	rv := requireSlice("slices.IsSortedFunc", x)
	for i := 1; i < rv.Len(); i++ {
		if cmp(rv.Index(i).Interface(), rv.Index(i-1).Interface()) < 0 {
			return false
		}
	}
	return true
}

// reflectSlicesBinarySearchFunc searches x for target using cmp, returning the insertion
// position and a found flag. Equivalent to slices.BinarySearchFunc.
//
// Takes x (any) which must be a slice, target (any) the search target (may be of a
// different type than the slice elements), and cmp (func(any, any) int) which compares an
// element to the target.
//
// Returns (index, found).
func reflectSlicesBinarySearchFunc(x, target any, cmp func(any, any) int) (int, bool) {
	rv := requireSlice("slices.BinarySearchFunc", x)
	lo, hi := 0, rv.Len()
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if cmp(rv.Index(mid).Interface(), target) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < rv.Len() && cmp(rv.Index(lo).Interface(), target) == 0 {
		return lo, true
	}
	return lo, false
}

// reflectSlicesCompareFunc compares s1 and s2 lexicographically using cmp. Equivalent to
// slices.CompareFunc.
//
// Takes s1 (any), s2 (any) both slices, and cmp (func(any, any) int).
//
// Returns the first non-zero cmp result, or 0 / +-1 by length.
func reflectSlicesCompareFunc(s1, s2 any, cmp func(any, any) int) int {
	r1 := requireSlice("slices.CompareFunc", s1)
	r2 := requireSlice("slices.CompareFunc", s2)
	minLen := min(r2.Len(), r1.Len())
	for i := range minLen {
		if c := cmp(r1.Index(i).Interface(), r2.Index(i).Interface()); c != 0 {
			return c
		}
	}
	switch {
	case r1.Len() < r2.Len():
		return -1
	case r1.Len() > r2.Len():
		return 1
	default:
		return 0
	}
}

// reflectSlicesEqualFunc reports whether s1 and s2 have equal length and all pairwise
// elements compare equal under eq. Equivalent to slices.EqualFunc.
//
// Takes s1 (any), s2 (any), eq (func(any, any) bool).
//
// Returns true when the slices are pairwise equal.
func reflectSlicesEqualFunc(s1, s2 any, eq func(any, any) bool) bool {
	r1 := requireSlice("slices.EqualFunc", s1)
	r2 := requireSlice("slices.EqualFunc", s2)
	if r1.Len() != r2.Len() {
		return false
	}
	for i := range r1.Len() {
		if !eq(r1.Index(i).Interface(), r2.Index(i).Interface()) {
			return false
		}
	}
	return true
}

// reflectMapsDeleteFunc removes every (k, v) pair from m for which del returns true.
// Equivalent to maps.DeleteFunc.
//
// Takes m (any) a map, and del (func(any, any) bool).
//
// Panics if m is not a map.
func reflectMapsDeleteFunc(m any, del func(any, any) bool) {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.DeleteFunc: unsupported type %T", m))
	}
	var toDelete []reflect.Value
	mapIter := rv.MapRange()
	for mapIter.Next() {
		if del(mapIter.Key().Interface(), mapIter.Value().Interface()) {
			toDelete = append(toDelete, mapIter.Key())
		}
	}
	for _, k := range toDelete {
		rv.SetMapIndex(k, reflect.Value{})
	}
}

// reflectMapsEqualFunc reports whether m1 and m2 have the same key set and equal values
// under eq. Equivalent to maps.EqualFunc.
//
// Takes m1 (any), m2 (any), eq (func(any, any) bool).
//
// Returns true if the maps are key-equal and value-equal under eq.
//
// Panics if either argument is not a map.
func reflectMapsEqualFunc(m1, m2 any, eq func(any, any) bool) bool {
	r1 := reflect.ValueOf(m1)
	r2 := reflect.ValueOf(m2)
	if r1.Kind() != reflect.Map || r2.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.EqualFunc: both arguments must be maps, got %T and %T", m1, m2))
	}
	if r1.Len() != r2.Len() {
		return false
	}
	mapIter := r1.MapRange()
	for mapIter.Next() {
		k := mapIter.Key()
		v := mapIter.Value()
		v2 := r2.MapIndex(k)
		if !v2.IsValid() {
			return false
		}
		if !eq(v.Interface(), v2.Interface()) {
			return false
		}
	}
	return true
}
