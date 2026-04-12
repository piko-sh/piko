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
	"hash/maphash"
	"iter"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func newSeedForTesting() maphash.Seed { return maphash.MakeSeed() }

func requireType[T any](t *testing.T, v any) T {
	t.Helper()
	out, ok := v.(T)
	require.True(t, ok, "expected %T", *new(T))
	return out
}

type customInt32 int32

type customString string

func TestReflectSlicesSortInt32(t *testing.T) {
	t.Parallel()
	in := []int32{3, 1, 4, 1, 5, 9, 2, 6}
	reflectSlicesSort(in)
	require.Equal(t, []int32{1, 1, 2, 3, 4, 5, 6, 9}, in)
}

func TestReflectSlicesSortNamedType(t *testing.T) {
	t.Parallel()
	in := []customInt32{3, 1, 4}
	reflectSlicesSort(in)
	require.Equal(t, []customInt32{1, 3, 4}, in)
}

func TestReflectSlicesSortFloat32(t *testing.T) {
	t.Parallel()
	in := []float32{3.5, 1.5, 4.5}
	reflectSlicesSort(in)
	require.Equal(t, []float32{1.5, 3.5, 4.5}, in)
}

func TestReflectSlicesMinMax(t *testing.T) {
	t.Parallel()
	require.Equal(t, int32(1), reflectSlicesMin([]int32{3, 1, 4, 1, 5}))
	require.Equal(t, int32(5), reflectSlicesMax([]int32{3, 1, 4, 1, 5}))
	require.Equal(t, customString("apple"), reflectSlicesMin([]customString{"cat", "apple", "bee"}))
	require.Equal(t, customString("cat"), reflectSlicesMax([]customString{"cat", "apple", "bee"}))
}

func TestReflectSlicesBinarySearch(t *testing.T) {
	t.Parallel()
	idx, found := reflectSlicesBinarySearch([]int32{1, 3, 5, 7, 9}, int32(5))
	require.Equal(t, 2, idx)
	require.True(t, found)

	idx, found = reflectSlicesBinarySearch([]int32{1, 3, 5, 7, 9}, int32(4))
	require.Equal(t, 2, idx)
	require.False(t, found)
}

func TestReflectSlicesIsSorted(t *testing.T) {
	t.Parallel()
	require.True(t, reflectSlicesIsSorted([]int32{1, 2, 3, 4}))
	require.False(t, reflectSlicesIsSorted([]int32{3, 1, 4}))
	require.True(t, reflectSlicesIsSorted([]float32{}))
}

func TestReflectSlicesCompare(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, reflectSlicesCompare([]int32{1, 2}, []int32{1, 2}))
	require.Equal(t, -1, reflectSlicesCompare([]int32{1, 2}, []int32{1, 3}))
	require.Equal(t, 1, reflectSlicesCompare([]int32{1, 3}, []int32{1, 2}))
	require.Equal(t, -1, reflectSlicesCompare([]int32{1}, []int32{1, 2}))
}

func TestReflectSlicesContainsEqualIndex(t *testing.T) {
	t.Parallel()
	require.True(t, reflectSlicesContains([]customString{"a", "b", "c"}, customString("b")))
	require.False(t, reflectSlicesContains([]customString{"a", "b", "c"}, customString("z")))
	require.True(t, reflectSlicesEqual([]customString{"a", "b"}, []customString{"a", "b"}))
	require.False(t, reflectSlicesEqual([]customString{"a"}, []customString{"a", "b"}))
	require.Equal(t, 1, reflectSlicesIndex([]customString{"a", "b", "c"}, customString("b")))
	require.Equal(t, -1, reflectSlicesIndex([]customString{"a", "b", "c"}, customString("z")))
}

func TestReflectCmpCompare(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, reflectCmpCompare(int32(5), int32(5)))
	require.Equal(t, -1, reflectCmpCompare(int32(3), int32(5)))
	require.Equal(t, 1, reflectCmpCompare(int32(7), int32(5)))
	require.Equal(t, -1, reflectCmpCompare(customString("a"), customString("b")))
}

func TestReflectCmpLess(t *testing.T) {
	t.Parallel()
	require.True(t, reflectCmpLess(int32(3), int32(5)))
	require.False(t, reflectCmpLess(int32(5), int32(3)))
	require.False(t, reflectCmpLess(int32(5), int32(5)))
}

func TestReflectCmpOr(t *testing.T) {
	t.Parallel()
	require.Equal(t, int32(0), reflectCmpOr(int32(0), int32(0), int32(0)))
	require.Equal(t, int32(5), reflectCmpOr(int32(0), int32(5), int32(7)))
	require.Equal(t, "first", reflectCmpOr("first", "second"))
	require.Nil(t, reflectCmpOr())
}

func TestWrappedUniqueMakePrimitives(t *testing.T) {
	t.Parallel()
	for _, value := range []any{
		"hello",
		int(42),
		int8(7),
		int16(7),
		int32(7),
		int64(7),
		uint(7),
		uint8(7),
		uint16(7),
		uint32(7),
		uint64(7),
		uintptr(7),
		float32(1.5),
		float64(1.5),
		complex64(complex(1, 2)),
		complex128(complex(1, 2)),
		true,
	} {
		result := wrappedUniqueMake(value)
		require.NotNil(t, result, "value=%T", value)
	}
}

func TestWrappedUniqueMakeRejectsNonComparable(t *testing.T) {
	t.Parallel()
	type customStruct struct{ Name string }
	require.PanicsWithValue(t,
		"unique.Make: unsupported type driven_system_symbols.customStruct (must be a built-in comparable)",
		func() { wrappedUniqueMake(customStruct{}) },
	)
}

func TestWrappedWeakMakePrimitives(t *testing.T) {
	t.Parallel()
	s := "hello"
	i := 42
	f := 1.5
	b := true
	arr := []byte{1, 2, 3}
	m := map[string]any{"k": "v"}
	for _, ptr := range []any{
		&s,
		&i,
		&f,
		&b,
		&arr,
		&m,
	} {
		result := wrappedWeakMake(ptr)
		require.NotNil(t, result, "ptr=%T", ptr)
	}
}

func TestWrappedWeakMakeRejectsNonPointer(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t,
		"weak.Make: argument must be a pointer, got int",
		func() { wrappedWeakMake(42) },
	)
}

func TestReflectCmpCompareNaN(t *testing.T) {
	t.Parallel()
	nan := float32(0)
	nan /= nan
	require.Equal(t, 0, reflectCmpCompare(nan, nan))
	require.Equal(t, -1, reflectCmpCompare(nan, float32(1)))
	require.Equal(t, 1, reflectCmpCompare(float32(1), nan))
}

func TestReflectSlicesClone(t *testing.T) {
	t.Parallel()
	original := []customString{"a", "b", "c"}
	cloned := requireType[[]customString](t, reflectSlicesClone(original))
	require.Equal(t, original, cloned)
	cloned[0] = "z"
	require.Equal(t, customString("a"), original[0])
}

func TestReflectSlicesReverse(t *testing.T) {
	t.Parallel()
	in := []int32{1, 2, 3, 4}
	reflectSlicesReverse(in)
	require.Equal(t, []int32{4, 3, 2, 1}, in)
	odd := []customString{"a", "b", "c"}
	reflectSlicesReverse(odd)
	require.Equal(t, []customString{"c", "b", "a"}, odd)
}

func TestReflectSlicesConcat(t *testing.T) {
	t.Parallel()
	result := requireType[[]int32](t, reflectSlicesConcat([]int32{1, 2}, []int32{3, 4}, []int32{5}))
	require.Equal(t, []int32{1, 2, 3, 4, 5}, result)
}

func TestReflectMapsClone(t *testing.T) {
	t.Parallel()
	original := map[customString]int32{"a": 1, "b": 2}
	cloned := requireType[map[customString]int32](t, reflectMapsClone(original))
	require.Equal(t, original, cloned)
	cloned["c"] = 3
	require.Equal(t, 2, len(original))
}

func TestReflectMapsCopy(t *testing.T) {
	t.Parallel()
	dst := map[customString]int32{"a": 1}
	src := map[customString]int32{"b": 2, "c": 3}
	reflectMapsCopy(dst, src)
	require.Equal(t, map[customString]int32{"a": 1, "b": 2, "c": 3}, dst)
}

func TestReflectMapsEqual(t *testing.T) {
	t.Parallel()
	require.True(t, reflectMapsEqual(
		map[customString]int32{"a": 1, "b": 2},
		map[customString]int32{"a": 1, "b": 2}))
	require.False(t, reflectMapsEqual(
		map[customString]int32{"a": 1},
		map[customString]int32{"a": 2}))
	require.False(t, reflectMapsEqual(
		map[customString]int32{"a": 1},
		map[customString]int32{"a": 1, "b": 2}))
}

func TestWrappedSyncOnceValue(t *testing.T) {
	t.Parallel()
	calls := 0
	producer := func() any {
		calls++
		return "memoised"
	}
	memoised := wrappedSyncOnceValue(producer)
	require.Equal(t, "memoised", memoised())
	require.Equal(t, "memoised", memoised())
	require.Equal(t, "memoised", memoised())
	require.Equal(t, 1, calls)
}

func TestWrappedMaphashComparable(t *testing.T) {
	t.Parallel()
	seed := newSeedForTesting()
	h1 := wrappedMaphashComparable(seed, customString("hello"))
	h2 := wrappedMaphashComparable(seed, customString("hello"))
	require.Equal(t, h1, h2)
	hDiff := wrappedMaphashComparable(seed, customString("world"))
	require.NotEqual(t, h1, hDiff)
}

func TestWrappedMaphashComparableNonBuiltin(t *testing.T) {
	t.Parallel()
	type point struct{ X, Y int }
	seed := newSeedForTesting()
	h1 := wrappedMaphashComparable(seed, point{1, 2})
	h2 := wrappedMaphashComparable(seed, point{1, 2})
	require.Equal(t, h1, h2)
}

func TestWrappedIterPull(t *testing.T) {
	t.Parallel()
	seq := func(yield func(int) bool) {
		for i := 1; i <= 5; i++ {
			if !yield(i) {
				return
			}
		}
	}
	nextAny, stopAny := wrappedIterPull(seq)
	next := requireType[func() (any, bool)](t, nextAny)
	stop := requireType[func()](t, stopAny)
	defer stop()

	var collected []int
	for {
		v, ok := next()
		if !ok {
			break
		}
		collected = append(collected, v.(int))
	}
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected)
}

func TestWrappedIterPullEarlyStop(t *testing.T) {
	t.Parallel()
	seq := func(yield func(int) bool) {
		for i := 1; i <= 100; i++ {
			if !yield(i) {
				return
			}
		}
	}
	nextAny, stopAny := wrappedIterPull(seq)
	next := requireType[func() (any, bool)](t, nextAny)
	stop := requireType[func()](t, stopAny)

	v, ok := next()
	require.True(t, ok)
	require.Equal(t, 1, v)
	stop()
}

func TestReflectSlicesAll(t *testing.T) {
	t.Parallel()
	source := []customString{"a", "b", "c"}
	seq := requireType[func(yield func(int, customString) bool)](t, reflectSlicesAll(source))
	var indices []int
	var values []customString
	for i, v := range seq {
		indices = append(indices, i)
		values = append(values, v)
	}
	require.Equal(t, []int{0, 1, 2}, indices)
	require.Equal(t, source, values)
}

func TestReflectSlicesBackward(t *testing.T) {
	t.Parallel()
	source := []customString{"a", "b", "c"}
	seq := requireType[func(yield func(int, customString) bool)](t, reflectSlicesBackward(source))
	var values []customString
	for _, v := range seq {
		values = append(values, v)
	}
	require.Equal(t, []customString{"c", "b", "a"}, values)
}

func TestReflectSlicesValues(t *testing.T) {
	t.Parallel()
	source := []customString{"a", "b", "c"}
	seq := requireType[func(yield func(customString) bool)](t, reflectSlicesValues(source))
	var values []customString
	for v := range seq {
		values = append(values, v)
	}
	require.Equal(t, source, values)
}

func TestReflectSlicesChunk(t *testing.T) {
	t.Parallel()
	source := []customString{"a", "b", "c", "d", "e"}
	seq := requireType[func(yield func([]customString) bool)](t, reflectSlicesChunk(source, 2))
	var chunks [][]customString
	for c := range seq {
		chunkCopy := make([]customString, len(c))
		copy(chunkCopy, c)
		chunks = append(chunks, chunkCopy)
	}
	require.Equal(t, [][]customString{{"a", "b"}, {"c", "d"}, {"e"}}, chunks)
}

func TestReflectSlicesAppendSeq(t *testing.T) {
	t.Parallel()
	base := []customString{"a"}
	seq := func(yield func(customString) bool) {
		for _, v := range []customString{"b", "c"} {
			if !yield(v) {
				return
			}
		}
	}
	result := requireType[[]customString](t, reflectSlicesAppendSeq(base, seq))
	require.Equal(t, []customString{"a", "b", "c"}, result)
}

func TestReflectSlicesCompact(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "a", "b", "c", "c", "c", "d"}
	result := requireType[[]customString](t, reflectSlicesCompact(in))
	require.Equal(t, []customString{"a", "b", "c", "d"}, result)
}

func TestReflectSlicesCompactFunc(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "A", "b", "B", "B"}
	caseInsensitive := func(a, b any) bool {

		s1 := string(a.(customString))
		s2 := string(b.(customString))
		if len(s1) == 0 || len(s2) == 0 {
			return s1 == s2
		}
		return s1[0]|0x20 == s2[0]|0x20
	}
	result := requireType[[]customString](t, reflectSlicesCompactFunc(in, caseInsensitive))
	require.Equal(t, []customString{"a", "b"}, result)
}

func TestReflectSlicesDelete(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b", "c", "d", "e"}
	result := requireType[[]customString](t, reflectSlicesDelete(in, 1, 3))
	require.Equal(t, []customString{"a", "d", "e"}, result)
}

func TestReflectSlicesDeleteFunc(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "B", "c", "D", "e"}
	upper := func(v any) bool {
		s := string(v.(customString))
		return len(s) > 0 && s[0] < 'a'
	}
	result := requireType[[]customString](t, reflectSlicesDeleteFunc(in, upper))
	require.Equal(t, []customString{"a", "c", "e"}, result)
}

func TestReflectSlicesGrow(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b"}
	result := requireType[[]customString](t, reflectSlicesGrow(in, 10))
	require.Equal(t, in, result)
	require.GreaterOrEqual(t, cap(result), len(in)+10)
}

func TestReflectSlicesInsert(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "d"}
	result := requireType[[]customString](t, reflectSlicesInsert(in, 1, customString("b"), customString("c")))
	require.Equal(t, []customString{"a", "b", "c", "d"}, result)
}

func TestReflectSlicesRepeat(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b"}
	result := requireType[[]customString](t, reflectSlicesRepeat(in, 3))
	require.Equal(t, []customString{"a", "b", "a", "b", "a", "b"}, result)
}

func TestReflectSlicesReplace(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b", "c", "d"}
	result := requireType[[]customString](t, reflectSlicesReplace(in, 1, 3, customString("X"), customString("Y"), customString("Z")))
	require.Equal(t, []customString{"a", "X", "Y", "Z", "d"}, result)
}

func TestReflectSlicesClip(t *testing.T) {
	t.Parallel()
	in := make([]customString, 2, 10)
	in[0], in[1] = "a", "b"
	result := requireType[[]customString](t, reflectSlicesClip(in))
	require.Equal(t, []customString{"a", "b"}, result)
	require.Equal(t, 2, cap(result))
}

func TestReflectSlicesContainsFunc(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b", "c"}
	require.True(t, reflectSlicesContainsFunc(in, func(v any) bool { return v.(customString) == "b" }))
	require.False(t, reflectSlicesContainsFunc(in, func(v any) bool { return v.(customString) == "z" }))
}

func TestReflectSlicesIndexFunc(t *testing.T) {
	t.Parallel()
	in := []customString{"a", "b", "c"}
	require.Equal(t, 1, reflectSlicesIndexFunc(in, func(v any) bool { return v.(customString) == "b" }))
	require.Equal(t, -1, reflectSlicesIndexFunc(in, func(v any) bool { return v.(customString) == "z" }))
}

func TestReflectSlicesEqualFunc(t *testing.T) {
	t.Parallel()
	eq := func(a, b any) bool { return a.(customString) == b.(customString) }
	require.True(t, reflectSlicesEqualFunc([]customString{"a"}, []customString{"a"}, eq))
	require.False(t, reflectSlicesEqualFunc([]customString{"a"}, []customString{"b"}, eq))
}

func TestReflectMapsAll(t *testing.T) {
	t.Parallel()
	m := map[customString]int32{"a": 1, "b": 2}
	seq := requireType[func(yield func(customString, int32) bool)](t, reflectMapsAll(m))
	count := 0
	for k, v := range seq {
		require.Contains(t, m, k)
		require.Equal(t, m[k], v)
		count++
	}
	require.Equal(t, 2, count)
}

func TestReflectMapsKeysValues(t *testing.T) {
	t.Parallel()
	m := map[customString]int32{"a": 1, "b": 2}
	keysSeq := requireType[func(yield func(customString) bool)](t, reflectMapsKeys(m))
	keySet := map[customString]struct{}{}
	for k := range keysSeq {
		keySet[k] = struct{}{}
	}
	require.Equal(t, map[customString]struct{}{"a": {}, "b": {}}, keySet)

	valuesSeq := requireType[func(yield func(int32) bool)](t, reflectMapsValues(m))
	valSet := map[int32]struct{}{}
	for v := range valuesSeq {
		valSet[v] = struct{}{}
	}
	require.Equal(t, map[int32]struct{}{1: {}, 2: {}}, valSet)
}

func TestReflectMapsInsert(t *testing.T) {
	t.Parallel()
	m := map[customString]int32{"a": 1}
	seq := func(yield func(customString, int32) bool) {
		entries := []struct {
			k customString
			v int32
		}{{"b", 2}, {"c", 3}}
		for _, e := range entries {
			if !yield(e.k, e.v) {
				return
			}
		}
	}
	reflectMapsInsert(m, seq)
	require.Equal(t, map[customString]int32{"a": 1, "b": 2, "c": 3}, m)
}

func TestReflectMapsDeleteFunc(t *testing.T) {
	t.Parallel()
	m := map[customString]int32{"a": 1, "b": 2, "c": 3, "d": 4}
	reflectMapsDeleteFunc(m, func(k, v any) bool { return v.(int32)%2 == 0 })
	require.Equal(t, map[customString]int32{"a": 1, "c": 3}, m)
}

func TestReflectMapsEqualFunc(t *testing.T) {
	t.Parallel()
	a := map[customString]int32{"a": 1, "b": 2}
	b := map[customString]int32{"a": 1, "b": 2}
	eq := func(v1, v2 any) bool { return v1.(int32) == v2.(int32) }
	require.True(t, reflectMapsEqualFunc(a, b, eq))
	b["b"] = 99
	require.False(t, reflectMapsEqualFunc(a, b, eq))
}

func TestWrappedIterPull2(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string, int) bool) {
		entries := []struct {
			k string
			v int
		}{{"a", 1}, {"b", 2}, {"c", 3}}
		for _, e := range entries {
			if !yield(e.k, e.v) {
				return
			}
		}
	}
	nextAny, stopAny := wrappedIterPull2(seq)
	next := requireType[func() (any, any, bool)](t, nextAny)
	stop := requireType[func()](t, stopAny)
	defer stop()

	keys := []string{}
	values := []int{}
	for {
		k, v, ok := next()
		if !ok {
			break
		}
		keys = append(keys, k.(string))
		values = append(values, v.(int))
	}
	require.Equal(t, []string{"a", "b", "c"}, keys)
	require.Equal(t, []int{1, 2, 3}, values)
}

func TestIterCollectAnyHomogeneousPromotesToConcreteSlice(t *testing.T) {
	t.Parallel()

	seq := iter.Seq[any](func(yield func(any) bool) {
		for _, v := range []any{1, 2, 3} {
			if !yield(v) {
				return
			}
		}
	})
	result := iterCollectAny(reflect.ValueOf(seq))
	require.Equal(t, []int{1, 2, 3}, result)
}

func TestIterCollectAnyNilFirstElementDoesNotPanic(t *testing.T) {
	t.Parallel()

	seq := iter.Seq[any](func(yield func(any) bool) {
		for _, v := range []any{nil, "after-nil"} {
			if !yield(v) {
				return
			}
		}
	})
	result := iterCollectAny(reflect.ValueOf(seq))
	require.Equal(t, []any{nil, "after-nil"}, result)
}

func TestIterCollectAnyHeterogeneousDoesNotPanic(t *testing.T) {
	t.Parallel()

	seq := iter.Seq[any](func(yield func(any) bool) {
		for _, v := range []any{1, "two", 3.0} {
			if !yield(v) {
				return
			}
		}
	})
	result := iterCollectAny(reflect.ValueOf(seq))
	require.Equal(t, []any{1, "two", 3.0}, result)
}

func TestIterCollectAnyEmptyReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	seq := iter.Seq[any](func(_ func(any) bool) {})
	result := iterCollectAny(reflect.ValueOf(seq))
	require.Equal(t, []any(nil), result)
}
