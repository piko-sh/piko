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

// makeIterSeq builds a reflect.Value of type iter.Seq[E].
//
// The synthesised value is func(yield func(E) bool); its body invokes producer for each
// yield iteration. The producer is called with the yield function as its only argument;
// it should call yield and respect its bool return to stop early.
//
// Takes elemType (reflect.Type) which is E.
// Takes producer (func(yield reflect.Value)) the body of the iter.Seq.
//
// Returns the synthesised iter.Seq value.
func makeIterSeq(elemType reflect.Type, producer func(yield reflect.Value)) reflect.Value {
	yieldType := reflect.FuncOf(
		[]reflect.Type{elemType},
		[]reflect.Type{reflect.TypeFor[bool]()},
		false,
	)
	seqType := reflect.FuncOf([]reflect.Type{yieldType}, nil, false)
	return reflect.MakeFunc(seqType, func(args []reflect.Value) []reflect.Value {
		producer(args[0])
		return nil
	})
}

// makeIterSeq2 builds a reflect.Value of type iter.Seq2[K, V] (i.e. func(yield func(K, V)
// bool)) whose body invokes producer for each yield iteration.
//
// Takes keyType (reflect.Type) which is K.
// Takes valType (reflect.Type) which is V.
// Takes producer (func(yield reflect.Value)) the body of the iter.Seq2.
//
// Returns the synthesised iter.Seq2 value.
func makeIterSeq2(keyType, valType reflect.Type, producer func(yield reflect.Value)) reflect.Value {
	yieldType := reflect.FuncOf(
		[]reflect.Type{keyType, valType},
		[]reflect.Type{reflect.TypeFor[bool]()},
		false,
	)
	seqType := reflect.FuncOf([]reflect.Type{yieldType}, nil, false)
	return reflect.MakeFunc(seqType, func(args []reflect.Value) []reflect.Value {
		producer(args[0])
		return nil
	})
}

// reflectSlicesAll returns an iter.Seq2[int, E] over (index, value) pairs of the slice.
// Equivalent to slices.All.
//
// Takes x (any) which must be a slice.
//
// Returns iter.Seq2[int, E] as any.
func reflectSlicesAll(x any) any {
	rv := requireSlice("slices.All", x)
	intType := reflect.TypeFor[int]()
	elemType := rv.Type().Elem()
	return makeIterSeq2(intType, elemType, func(yield reflect.Value) {
		for i := range rv.Len() {
			results := yield.Call([]reflect.Value{reflect.ValueOf(i), rv.Index(i)})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectSlicesBackward returns an iter.Seq2[int, E] iterating in reverse order.
// Equivalent to slices.Backward.
//
// Takes x (any) which must be a slice.
//
// Returns iter.Seq2[int, E] as any.
func reflectSlicesBackward(x any) any {
	rv := requireSlice("slices.Backward", x)
	intType := reflect.TypeFor[int]()
	elemType := rv.Type().Elem()
	return makeIterSeq2(intType, elemType, func(yield reflect.Value) {
		for i := rv.Len() - 1; i >= 0; i-- {
			results := yield.Call([]reflect.Value{reflect.ValueOf(i), rv.Index(i)})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectSlicesValues returns an iter.Seq[E] over the slice's elements. Equivalent to
// slices.Values.
//
// Takes x (any) which must be a slice.
//
// Returns iter.Seq[E] as any.
func reflectSlicesValues(x any) any {
	rv := requireSlice("slices.Values", x)
	elemType := rv.Type().Elem()
	return makeIterSeq(elemType, func(yield reflect.Value) {
		for i := range rv.Len() {
			results := yield.Call([]reflect.Value{rv.Index(i)})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectSlicesChunk returns an iter.Seq[S] yielding contiguous sub-slices of length up
// to n. Equivalent to slices.Chunk.
//
// Takes x (any) which must be a slice, and n (int) which must be >= 1.
//
// Returns iter.Seq[S] as any.
//
// Panics if x is not a slice or n < 1.
func reflectSlicesChunk(x any, n int) any {
	rv := requireSlice("slices.Chunk", x)
	if n < 1 {
		panic(fmt.Sprintf("slices.Chunk: n must be >= 1, got %d", n))
	}
	sliceType := rv.Type()
	return makeIterSeq(sliceType, func(yield reflect.Value) {
		length := rv.Len()
		for i := 0; i < length; i += n {
			end := min(i+n, length)
			results := yield.Call([]reflect.Value{rv.Slice(i, end)})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectSlicesAppendSeq appends every value yielded by seq onto s and returns the
// (possibly grown) slice. Equivalent to slices.AppendSeq.
//
// Takes s (any) which must be a slice, and seq (any) which must be an iter.Seq[E] over
// compatible element type.
//
// Returns the resulting slice.
func reflectSlicesAppendSeq(s, seq any) any {
	rv := requireSlice("slices.AppendSeq", s)
	result := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
	reflect.Copy(result, rv)
	seqVal := reflect.ValueOf(seq)
	for v := range seqVal.Seq() {
		result = reflect.Append(result, v)
	}
	return result.Interface()
}

// reflectMapsAll returns an iter.Seq2[K, V] over the map's key-value pairs. Equivalent to
// maps.All.
//
// Takes m (any) which must be a map.
//
// Returns iter.Seq2[K, V] as any.
//
// Panics if m is not a map.
func reflectMapsAll(m any) any {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.All: unsupported type %T", m))
	}
	keyType := rv.Type().Key()
	valType := rv.Type().Elem()
	return makeIterSeq2(keyType, valType, func(yield reflect.Value) {
		mapIter := rv.MapRange()
		for mapIter.Next() {
			results := yield.Call([]reflect.Value{mapIter.Key(), mapIter.Value()})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectMapsKeys returns an iter.Seq[K] over the map's keys. Equivalent to maps.Keys.
//
// Takes m (any) which must be a map.
//
// Returns iter.Seq[K] as any.
//
// Panics if m is not a map.
func reflectMapsKeys(m any) any {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.Keys: unsupported type %T", m))
	}
	keyType := rv.Type().Key()
	return makeIterSeq(keyType, func(yield reflect.Value) {
		mapIter := rv.MapRange()
		for mapIter.Next() {
			results := yield.Call([]reflect.Value{mapIter.Key()})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectMapsValues returns an iter.Seq[V] over the map's values. Equivalent to
// maps.Values.
//
// Takes m (any) which must be a map.
//
// Returns iter.Seq[V] as any.
//
// Panics if m is not a map.
func reflectMapsValues(m any) any {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.Values: unsupported type %T", m))
	}
	valType := rv.Type().Elem()
	return makeIterSeq(valType, func(yield reflect.Value) {
		mapIter := rv.MapRange()
		for mapIter.Next() {
			results := yield.Call([]reflect.Value{mapIter.Value()})
			if !results[0].Bool() {
				return
			}
		}
	}).Interface()
}

// reflectMapsInsert consumes an iter.Seq2[K, V] and writes every pair into m. Equivalent
// to maps.Insert.
//
// Takes m (any) which must be a mutable map.
// Takes seq (any) which must be an iter.Seq2[K, V] of compatible key/value types.
//
// Panics if m is not a map.
func reflectMapsInsert(m, seq any) {
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("maps.Insert: unsupported type %T", m))
	}
	seqVal := reflect.ValueOf(seq)
	for k, v := range seqVal.Seq2() {
		rv.SetMapIndex(k, v)
	}
}
