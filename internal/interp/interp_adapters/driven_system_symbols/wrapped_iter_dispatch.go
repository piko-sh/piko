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
	"iter"
	"maps"
	"reflect"
	"slices"
	"sort"
)

const (
	// pkgSlices is the Symbols map key for the slices package.
	pkgSlices = "slices"

	// pkgMaps is the Symbols map key for the maps package.
	pkgMaps = "maps"

	// minSequence2YieldParams is the minimum number of parameters for a
	// valid iter.Seq2 yield function (key and value).
	minSequence2YieldParams = 2
)

// wrappedSlicesCollect wraps slices.Collect for use with dynamic iterator
// types in the interpreter.
//
// Takes sequence (any) which is the iterator to collect into a slice.
//
// Returns a typed slice containing all values yielded by the iterator.
func wrappedSlicesCollect(sequence any) any {
	sequenceVal := reflect.ValueOf(sequence)
	elemType, ok := iterSequenceElemType(sequenceVal)
	if !ok {
		return iterCollectAny(sequenceVal)
	}
	sequence = convertSequenceToUnnamed(sequenceVal, elemType)

	switch elemType {
	case reflect.TypeFor[string]():
		return slices.Collect(sequence.(func(func(string) bool)))
	case reflect.TypeFor[int]():
		return slices.Collect(sequence.(func(func(int) bool)))
	case reflect.TypeFor[int64]():
		return slices.Collect(sequence.(func(func(int64) bool)))
	case reflect.TypeFor[float64]():
		return slices.Collect(sequence.(func(func(float64) bool)))
	case reflect.TypeFor[byte]():
		return slices.Collect(sequence.(func(func(byte) bool)))
	case reflect.TypeFor[bool]():
		return slices.Collect(sequence.(func(func(bool) bool)))
	default:
		return iterCollectReflect(sequenceVal, elemType)
	}
}

// wrappedSlicesSorted wraps slices.Sorted for use with dynamic iterator types
// in the interpreter.
//
// Takes sequence (any) which is the iterator to collect and sort.
//
// Returns a sorted typed slice containing all values yielded by the iterator.
func wrappedSlicesSorted(sequence any) any {
	sequenceVal := reflect.ValueOf(sequence)
	elemType, ok := iterSequenceElemType(sequenceVal)
	if !ok {
		result := iterCollectAny(sequenceVal)
		reflectSortOrdered(result)
		return result
	}
	sequence = convertSequenceToUnnamed(sequenceVal, elemType)

	switch elemType {
	case reflect.TypeFor[string]():
		return slices.Sorted(sequence.(func(func(string) bool)))
	case reflect.TypeFor[int]():
		return slices.Sorted(sequence.(func(func(int) bool)))
	case reflect.TypeFor[int64]():
		return slices.Sorted(sequence.(func(func(int64) bool)))
	case reflect.TypeFor[float64]():
		return slices.Sorted(sequence.(func(func(float64) bool)))
	case reflect.TypeFor[byte]():
		return slices.Sorted(sequence.(func(func(byte) bool)))
	default:
		result := iterCollectReflect(sequenceVal, elemType)
		reflectSortOrdered(result)
		return result
	}
}

// wrappedSlicesSortedFunc wraps slices.SortedFunc for use with dynamic
// iterator types in the interpreter.
//
// Takes sequence (any) which is the iterator to collect and sort.
// Takes compareFunction (func(any, any) int) which is the comparison function.
//
// Returns a sorted slice collected from the iterator using compareFunction.
func wrappedSlicesSortedFunc(sequence any, compareFunction func(any, any) int) any {
	return sortedFuncCommon(sequence, compareFunction, false)
}

// wrappedSlicesSortedStableFunc wraps slices.SortedStableFunc for use with
// dynamic iterator types in the interpreter.
//
// Takes sequence (any) which is the iterator to collect and sort.
// Takes compareFunction (func(any, any) int) which is the comparison function.
//
// Returns a stably sorted slice collected from the iterator using compareFunction.
func wrappedSlicesSortedStableFunc(sequence any, compareFunction func(any, any) int) any {
	return sortedFuncCommon(sequence, compareFunction, true)
}

// sortedFuncCommon implements both SortedFunc and SortedStableFunc.
//
// Takes sequence (any) which is the iterator to collect and sort.
// Takes compareFunction (func(any, any) int) which is the comparison function for
// ordering.
// Takes stable (bool) which selects between stable and unstable sort.
//
// Returns a sorted slice collected from the iterator.
func sortedFuncCommon(sequence any, compareFunction func(any, any) int, stable bool) any {
	sequenceVal := reflect.ValueOf(sequence)
	elemType, ok := iterSequenceElemType(sequenceVal)
	if !ok {
		result := iterCollectAny(sequenceVal)
		reflectSortWithCompareFunction(result, compareFunction)
		return result
	}
	sequence = convertSequenceToUnnamed(sequenceVal, elemType)

	switch elemType {
	case reflect.TypeFor[string]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[string], slices.SortedStableFunc[string])
	case reflect.TypeFor[int]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[int], slices.SortedStableFunc[int])
	case reflect.TypeFor[int64]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[int64], slices.SortedStableFunc[int64])
	case reflect.TypeFor[float64]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[float64], slices.SortedStableFunc[float64])
	case reflect.TypeFor[byte]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[byte], slices.SortedStableFunc[byte])
	case reflect.TypeFor[bool]():
		return sortedFuncDispatch(sequence, compareFunction, stable, slices.SortedFunc[bool], slices.SortedStableFunc[bool])
	default:
		result := iterCollectReflect(sequenceVal, elemType)
		reflectSortWithCompareFunction(result, compareFunction)
		return result
	}
}

// wrappedMapsCollect wraps maps.Collect for use with dynamic iterator types in
// the interpreter.
//
// Takes sequence (any) which is the Seq2 iterator to collect into a map.
//
// Returns a typed map containing all key-value pairs yielded by the iterator.
func wrappedMapsCollect(sequence any) any {
	sequenceVal := reflect.ValueOf(sequence)
	keyType, valType, ok := iterSequence2Types(sequenceVal)
	if !ok {
		return iterCollectAny2(sequenceVal)
	}
	sequence = convertSequence2ToUnnamed(sequenceVal, keyType, valType)

	if result, matched := mapsCollectByType(sequence, keyType, valType); matched {
		return result
	}
	return iterCollectSequence2Reflect(sequenceVal, keyType, valType)
}

// mapsCollectByType dispatches maps.Collect to the correct concrete
// key/value type combination.
//
// Takes sequence (any) which is the iterator sequence to collect.
// Takes keyType (reflect.Type) which is the map key type.
// Takes valType (reflect.Type) which is the map value type.
//
// Returns the collected map and true if the type combination is supported.
func mapsCollectByType(sequence any, keyType, valType reflect.Type) (any, bool) {
	switch keyType {
	case reflect.TypeFor[string]():
		return mapsCollectByVal[string](sequence, valType)
	case reflect.TypeFor[int]():
		return mapsCollectByVal[int](sequence, valType)
	default:
		return nil, false
	}
}

// mapsCollectByVal dispatches maps.Collect for a known key type K,
// switching on the value type at runtime.
//
// Takes sequence (any) which is the iterator sequence to collect.
// Takes valType (reflect.Type) which is the map value type.
//
// Returns the collected map and true if the value type is supported.
func mapsCollectByVal[K comparable](sequence any, valType reflect.Type) (any, bool) {
	switch valType {
	case reflect.TypeFor[string]():
		return mapsCollectTyped[K, string](sequence)
	case reflect.TypeFor[int]():
		return mapsCollectTyped[K, int](sequence)
	case reflect.TypeFor[float64]():
		return mapsCollectTyped[K, float64](sequence)
	case reflect.TypeFor[bool]():
		return mapsCollectTyped[K, bool](sequence)
	case reflect.TypeFor[any]():
		return mapsCollectTyped[K, any](sequence)
	default:
		return nil, false
	}
}

// mapsCollectTyped performs the typed maps.Collect call for a
// concrete key/value pair.
//
// Takes sequence (any) which is the iterator sequence to collect.
//
// Returns the collected map and true if the type assertion succeeds.
func mapsCollectTyped[K comparable, V any](sequence any) (any, bool) {
	typedSequence, ok := sequence.(iter.Seq2[K, V])
	if !ok {
		return nil, false
	}
	return maps.Collect(typedSequence), true
}

// iterSequenceElemType extracts the element type from an iter.Seq function via
// reflection.
//
// Takes sequenceVal (reflect.Value) which is the iterator function to inspect.
//
// Returns the element type and true if the yield function has at least one
// parameter.
//
// Panics if sequenceVal is not a single-argument function.
func iterSequenceElemType(sequenceVal reflect.Value) (reflect.Type, bool) {
	sequenceType := sequenceVal.Type()
	if sequenceType.Kind() != reflect.Func || sequenceType.NumIn() != 1 {
		panic(fmt.Sprintf("slices.Collect: expected iterator func, got %s", sequenceType))
	}
	yieldType := sequenceType.In(0)
	if yieldType.Kind() == reflect.Func && yieldType.NumIn() >= 1 {
		return yieldType.In(0), true
	}
	return nil, false
}

// iterSequence2Types extracts the key and value types from an iter.Seq2 function
// via reflection.
//
// Takes sequenceVal (reflect.Value) which is the Seq2 iterator
// function to inspect.
//
// Returns the key type, value type, and true if the yield function has at
// least two parameters.
//
// Panics if sequenceVal is not a single-argument function.
func iterSequence2Types(sequenceVal reflect.Value) (keyType, valType reflect.Type, ok bool) {
	sequenceType := sequenceVal.Type()
	if sequenceType.Kind() != reflect.Func || sequenceType.NumIn() != 1 {
		panic(fmt.Sprintf("maps.Collect: expected iterator func, got %s", sequenceType))
	}
	yieldType := sequenceType.In(0)
	if yieldType.Kind() == reflect.Func && yieldType.NumIn() >= minSequence2YieldParams {
		return yieldType.In(0), yieldType.In(1), true
	}
	return nil, nil, false
}

// convertSequenceToUnnamed converts a named iter.Seq value to an unnamed function
// type for type assertions.
//
// Takes sequenceVal (reflect.Value) which is the iter.Seq value to convert.
// Takes elemType (reflect.Type) which is the element type of the iterator.
//
// Returns the converted function as an unnamed func(func(E) bool) type.
func convertSequenceToUnnamed(sequenceVal reflect.Value, elemType reflect.Type) any {
	yieldType := reflect.FuncOf([]reflect.Type{elemType}, []reflect.Type{reflect.TypeFor[bool]()}, false)
	iterType := reflect.FuncOf([]reflect.Type{yieldType}, nil, false)
	return sequenceVal.Convert(iterType).Interface()
}

// convertSequence2ToUnnamed converts a named iter.Seq2 value to an unnamed
// function type for type assertions.
//
// Takes sequenceVal (reflect.Value) which is the iter.Seq2 value to convert.
// Takes keyType (reflect.Type) which is the key type of the iterator.
// Takes valType (reflect.Type) which is the value type of the iterator.
//
// Returns the converted function as an unnamed func(func(K, V) bool) type.
func convertSequence2ToUnnamed(sequenceVal reflect.Value, keyType, valType reflect.Type) any {
	yieldType := reflect.FuncOf([]reflect.Type{keyType, valType}, []reflect.Type{reflect.TypeFor[bool]()}, false)
	iterType := reflect.FuncOf([]reflect.Type{yieldType}, nil, false)
	return sequenceVal.Convert(iterType).Interface()
}

// iterCollectAny collects all values from an iterator into a typed slice using
// runtime type inference.
//
// Takes sequenceVal (reflect.Value) which is the iterator
// function to collect from.
//
// Returns a typed slice of the collected values, or []any if the
// slice is empty.
func iterCollectAny(sequenceVal reflect.Value) any {
	var collected []any
	yieldFunction := func(v any) bool {
		collected = append(collected, v)
		return true
	}
	sequenceVal.Call([]reflect.Value{reflect.ValueOf(yieldFunction)})
	if len(collected) == 0 {
		return collected
	}
	elemType := reflect.TypeOf(collected[0])
	result := reflect.MakeSlice(reflect.SliceOf(elemType), len(collected), len(collected))
	for i, v := range collected {
		result.Index(i).Set(reflect.ValueOf(v))
	}
	return result.Interface()
}

// iterCollectAny2 collects all key-value pairs from a Seq2 iterator into a
// map.
//
// Takes sequenceVal (reflect.Value) which is the Seq2 iterator
// function to collect from.
//
// Returns a map[any]any containing all collected key-value pairs.
func iterCollectAny2(sequenceVal reflect.Value) map[any]any {
	result := make(map[any]any)
	yieldFunction := func(k, v any) bool {
		result[k] = v
		return true
	}
	sequenceVal.Call([]reflect.Value{reflect.ValueOf(yieldFunction)})
	return result
}

// iterCollectReflect collects values from an iterator into a typed slice using
// reflect.
//
// Takes sequenceVal (reflect.Value) which is the iterator
// function to collect from.
// Takes elemType (reflect.Type) which is the element type of the
// resulting slice.
//
// Returns a typed slice of all values yielded by the iterator.
func iterCollectReflect(sequenceVal reflect.Value, elemType reflect.Type) any {
	sliceType := reflect.SliceOf(elemType)
	result := reflect.MakeSlice(sliceType, 0, 0)
	yieldType := sequenceVal.Type().In(0)
	yieldFunction := reflect.MakeFunc(yieldType, func(arguments []reflect.Value) []reflect.Value {
		result = reflect.Append(result, arguments[0])
		return []reflect.Value{reflect.ValueOf(true)}
	})
	sequenceVal.Call([]reflect.Value{yieldFunction})
	return result.Interface()
}

// iterCollectSequence2Reflect collects key-value pairs from a Seq2 iterator into a
// typed map using reflect.
//
// Takes sequenceVal (reflect.Value) which is the Seq2 iterator
// function to collect.
// Takes keyType (reflect.Type) which is the key type of the resulting map.
// Takes valType (reflect.Type) which is the value type of the resulting map.
//
// Returns a typed map containing all key-value pairs yielded by the iterator.
func iterCollectSequence2Reflect(sequenceVal reflect.Value, keyType, valType reflect.Type) any {
	mapType := reflect.MapOf(keyType, valType)
	result := reflect.MakeMap(mapType)
	yieldType := sequenceVal.Type().In(0)
	yieldFunction := reflect.MakeFunc(yieldType, func(arguments []reflect.Value) []reflect.Value {
		result.SetMapIndex(arguments[0], arguments[1])
		return []reflect.Value{reflect.ValueOf(true)}
	})
	sequenceVal.Call([]reflect.Value{yieldFunction})
	return result.Interface()
}

// sortedFuncDispatch calls the stable or unstable SortedFunc variant for a
// concrete element type E.
//
// Takes sequence (any) which is the iterator function to sort.
// Takes compareFunction (func(any, any) int) which is the comparison function.
// Takes stable (bool) which selects between stable and unstable sort.
// Takes unstableFunction (func) which is the unstable sort function.
// Takes stableFunction (func) which is the stable sort function.
//
// Returns a sorted slice collected from the iterator.
//
// Panics if sequence cannot be asserted to the expected function type.
func sortedFuncDispatch[E any](sequence any, compareFunction func(any, any) int, stable bool,
	unstableFunction, stableFunction func(iter.Seq[E], func(E, E) int) []E,
) any {
	rawSequence, ok := sequence.(func(func(E) bool))
	if !ok {
		panic(fmt.Sprintf("sortedFuncDispatch: expected func(func(%T) bool), got %T", *new(E), sequence))
	}
	typedSequence := iter.Seq[E](rawSequence)
	wrap := func(a, b E) int { return compareFunction(a, b) }
	if stable {
		return stableFunction(typedSequence, wrap)
	}
	return unstableFunction(typedSequence, wrap)
}

// reflectSortWithCompareFunction sorts a reflect-created slice using a comparison
// function that operates on any values.
//
// Takes slice (any) which is the slice to sort in place.
// Takes compareFunction (func(any, any) int) which is the comparison function for
// ordering.
func reflectSortWithCompareFunction(slice any, compareFunction func(any, any) int) {
	rv := reflect.ValueOf(slice)
	sort.SliceStable(slice, func(i, j int) bool { //nolint:revive // reflect-based sort
		return compareFunction(rv.Index(i).Interface(), rv.Index(j).Interface()) < 0
	})
}

// reflectSortOrdered sorts a reflect-created slice of ordered elements using
// their natural ordering.
//
// Takes slice (any) which is the slice to sort in place.
//
// Panics if the slice type is not a supported ordered type.
func reflectSortOrdered(slice any) {
	switch s := slice.(type) {
	case []string:
		slices.Sort(s)
	case []int:
		slices.Sort(s)
	case []int8:
		slices.Sort(s)
	case []int16:
		slices.Sort(s)
	case []int32:
		slices.Sort(s)
	case []int64:
		slices.Sort(s)
	case []uint:
		slices.Sort(s)
	case []uint8:
		slices.Sort(s)
	case []uint16:
		slices.Sort(s)
	case []uint32:
		slices.Sort(s)
	case []uint64:
		slices.Sort(s)
	case []float32:
		slices.Sort(s)
	case []float64:
		slices.Sort(s)
	default:
		panic(fmt.Sprintf("slices.Sorted: unsupported slice type %T", slice))
	}
}

func init() {
	Symbols[pkgSlices]["Collect"] = reflect.ValueOf(wrappedSlicesCollect)
	Symbols[pkgSlices]["Sorted"] = reflect.ValueOf(wrappedSlicesSorted)
	Symbols[pkgSlices]["SortedFunc"] = reflect.ValueOf(wrappedSlicesSortedFunc)
	Symbols[pkgSlices]["SortedStableFunc"] = reflect.ValueOf(wrappedSlicesSortedStableFunc)
	Symbols[pkgMaps]["Collect"] = reflect.ValueOf(wrappedMapsCollect)
}
