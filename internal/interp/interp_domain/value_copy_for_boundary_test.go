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
	"testing"

	"github.com/stretchr/testify/require"
)

type boundaryCopyTestStruct struct {
	N int
}

func TestValueCopyForBoundaryStructProducesIndependentStorage(t *testing.T) {
	t.Parallel()

	original := reflect.New(reflect.TypeFor[boundaryCopyTestStruct]()).Elem()
	original.Field(0).SetInt(7)

	snapshot := valueCopyForBoundary(original)
	require.True(t, snapshot.IsValid())
	require.Equal(t, reflect.Struct, snapshot.Kind(),
		"struct kind must be preserved when crossing a boundary copy")

	original.Field(0).SetInt(99)
	require.Equal(t, int64(7), snapshot.Field(0).Int(),
		"the copy must hold an independent struct value so mutating the source does not leak into the destination; this is the central guarantee of valueCopyForBoundary for value-typed locals across function args, assignments, and channel receives")
}

func TestValueCopyForBoundaryArrayProducesIndependentStorage(t *testing.T) {
	t.Parallel()

	original := reflect.New(reflect.TypeFor[[3]int]()).Elem()
	original.Index(0).SetInt(1)
	original.Index(1).SetInt(2)
	original.Index(2).SetInt(3)

	snapshot := valueCopyForBoundary(original)
	require.True(t, snapshot.IsValid())
	require.Equal(t, reflect.Array, snapshot.Kind())
	require.Equal(t, 3, snapshot.Len())

	original.Index(0).SetInt(99)
	require.Equal(t, int64(1), snapshot.Index(0).Int(),
		"the array copy must own its own backing storage, because Go's range-over-array spec requires the iteration to observe a snapshot of the array at loop entry, and the same independence is needed at every boundary crossing where Go's value semantics would copy")
}

func TestValueCopyForBoundaryPointerSharesPointee(t *testing.T) {
	t.Parallel()

	target := boundaryCopyTestStruct{N: 5}
	original := reflect.ValueOf(&target)

	snapshot := valueCopyForBoundary(original)
	require.True(t, snapshot.IsValid())
	require.Equal(t, reflect.Pointer, snapshot.Kind())
	require.Equal(t, original.Pointer(), snapshot.Pointer(),
		"pointer kind must pass through valueCopyForBoundary unchanged so the copy and source point at the same memory, because Go's pointer semantics share the pointee, and copying the pointer header is the right semantic at every boundary where a pointer is value-passed")
}

func TestValueCopyForBoundarySliceSharesUnderlyingArray(t *testing.T) {
	t.Parallel()

	source := []int{1, 2, 3}
	original := reflect.ValueOf(source)

	snapshot := valueCopyForBoundary(original)
	require.True(t, snapshot.IsValid())
	require.Equal(t, reflect.Slice, snapshot.Kind())
	require.Equal(t, 3, snapshot.Len())

	source[1] = 99
	require.Equal(t, int64(99), snapshot.Index(1).Int(),
		"slice header pass-through is the correct Go semantic: the slice value carries (ptr, len, cap), and copying it gives a second header pointing at the same backing array, exactly Go's behaviour for `s2 := s1` where s1 is a slice")
}

func TestValueCopyForBoundaryInvalidPassesThrough(t *testing.T) {
	t.Parallel()

	original := reflect.Value{}
	snapshot := valueCopyForBoundary(original)
	require.False(t, snapshot.IsValid(),
		"invalid (zero) reflect.Values must pass through unchanged so callers handing the helper a typed-nil-error interface or an uninitialised register do not allocate fresh storage for the no-op")
}

func TestSnapshotModeForStructAlwaysSnapshot(t *testing.T) {
	t.Parallel()

	structType := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "N", types.Typ[types.Int], false),
	}, nil)
	require.Equal(t, snapshotAlways, snapshotModeFor(structType),
		"struct types must always snapshot; without this the compiler would alias struct values across the move and the receiver could see post-move mutations from the sender")
}

func TestSnapshotModeForArrayAlwaysSnapshot(t *testing.T) {
	t.Parallel()

	arrayType := types.NewArray(types.Typ[types.Int], 4)
	require.Equal(t, snapshotAlways, snapshotModeFor(arrayType),
		"array types match struct semantics, because Go copies the entire array at value-pass boundaries, so snapshotAlways is the only correct answer")
}

func TestSnapshotModeForPointerNeverSnapshot(t *testing.T) {
	t.Parallel()

	pointerType := types.NewPointer(types.Typ[types.Int])
	require.Equal(t, snapshotNever, snapshotModeFor(pointerType),
		"pointer types must take the alias-fast path because Go's reference semantics for pointers permit aliasing, so the reflect.Value header copy already gives the correct behaviour")
}

func TestSnapshotModeForSliceNeverSnapshot(t *testing.T) {
	t.Parallel()

	sliceType := types.NewSlice(types.Typ[types.Int])
	require.Equal(t, snapshotNever, snapshotModeFor(sliceType),
		"slice types must take the alias-fast path, because Go slice headers are 24-byte structs whose copy semantics are already alias-correct via reflect.Value header copy")
}

func TestSnapshotModeForMapNeverSnapshot(t *testing.T) {
	t.Parallel()

	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	require.Equal(t, snapshotNever, snapshotModeFor(mapType),
		"map types must take the alias-fast path, because Go map references already alias their hash tables and reflect.Value's header copy preserves that semantic")
}

func TestSnapshotModeForChanNeverSnapshot(t *testing.T) {
	t.Parallel()

	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	require.Equal(t, snapshotNever, snapshotModeFor(chanType),
		"channel types must take the alias-fast path, because channel handles are inherently reference-typed and the helper's runtime kind switch would be wasted work for every channel-typed register move")
}

func TestSnapshotModeForBasicNeverSnapshot(t *testing.T) {
	t.Parallel()

	require.Equal(t, snapshotNever, snapshotModeFor(types.Typ[types.Int]),
		"basic int boxed into general must take alias-fast, because a header copy preserves the integer value byte-for-byte; no kind switch needed")
	require.Equal(t, snapshotNever, snapshotModeFor(types.Typ[types.String]),
		"basic string boxed into general must take alias-fast for the same reason: string headers in reflect.Value are alias-correct")
}

func TestSnapshotModeForInterfaceDynamic(t *testing.T) {
	t.Parallel()

	emptyInterface := types.NewInterfaceType(nil, nil).Complete()
	require.Equal(t, snapshotDynamic, snapshotModeFor(emptyInterface),
		"empty interface must classify dynamic, because at runtime it could hold a struct value and the snapshot decision can only be made then, so the helper's kind switch must remain in the dispatch path")
}

func TestSnapshotModeForNamedRecursesIntoUnderlying(t *testing.T) {
	t.Parallel()

	structType := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "N", types.Typ[types.Int], false),
	}, nil)
	pkg := types.NewPackage("test", "test")
	typeName := types.NewTypeName(0, pkg, "Wide", nil)
	namedType := types.NewNamed(typeName, structType, nil)

	require.Equal(t, snapshotAlways, snapshotModeFor(namedType),
		"named struct types must classify by their underlying struct kind; without this every user-defined struct would fall to dynamic, defeating the alias elision for the most common compile-time-known case")
}

func TestSnapshotModeForNilDynamic(t *testing.T) {
	t.Parallel()

	require.Equal(t, snapshotDynamic, snapshotModeFor(nil),
		"nil types must classify dynamic so that emitMoveTyped sites without static type info preserve current behaviour byte-for-byte, so passing nil from a not-yet-threaded call site must remain correct")
}

func TestGeneralMoveModeForMapsToMoveGeneralModeConstants(t *testing.T) {
	t.Parallel()

	pointerType := types.NewPointer(types.Typ[types.Int])
	require.Equal(t, moveGeneralModeAlias, generalMoveModeFor(pointerType),
		"alias-safe types must produce the moveGeneralModeAlias byte so the runtime takes the direct register copy without invoking valueCopyForBoundary")

	structType := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "N", types.Typ[types.Int], false),
	}, nil)
	require.Equal(t, moveGeneralModeSnapshot, generalMoveModeFor(structType),
		"struct types must produce the moveGeneralModeSnapshot byte so the runtime always invokes copyReflectValue without a kind switch")

	require.Equal(t, moveGeneralModeDynamic, generalMoveModeFor(nil),
		"nil types must produce the moveGeneralModeDynamic byte so handleMoveGeneral falls through to valueCopyForBoundary's runtime kind switch, which preserves correctness for sites without static type info")
}
