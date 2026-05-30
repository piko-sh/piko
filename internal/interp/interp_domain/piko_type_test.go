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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPikoTypeNamedInterface(t *testing.T) {
	t.Parallel()
	pt := newPikoTypeNamedInterface("github.com/foo/bar", "MyIface", []string{"Foo", "Bar"})
	require.Equal(t, "MyIface", pt.Name())
	require.Equal(t, "bar.MyIface", pt.String())
	require.Equal(t, "github.com/foo/bar", pt.PkgPath())
	require.Equal(t, reflect.Interface, pt.Kind())
	require.Equal(t, 2, pt.NumMethod())
	require.Equal(t, "Foo", pt.Method(0).Name)
	require.Equal(t, "Bar", pt.Method(1).Name)
	method, ok := pt.MethodByName("Foo")
	require.True(t, ok)
	require.Equal(t, "Foo", method.Name)
	_, ok = pt.MethodByName("Missing")
	require.False(t, ok)
}

func TestPikoTypeNamedInterfaceNoPackage(t *testing.T) {
	t.Parallel()
	pt := newPikoTypeNamedInterface("", "error", []string{"Error"})
	require.Equal(t, "error", pt.Name())
	require.Equal(t, "error", pt.String())
}

func TestPikoTypePointerToInterface(t *testing.T) {
	t.Parallel()
	inner := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	ptr := pikoTypeOfPointer(inner)
	require.Equal(t, "*main.myiface", ptr.String())
	require.Equal(t, "", ptr.Name(), "pointer types have empty Name (Go convention)")
	require.Equal(t, reflect.Pointer, ptr.Kind())
	require.Equal(t, "main.myiface", ptr.Elem().String())
	require.Equal(t, reflect.Interface, ptr.Elem().Kind())
}

func TestPikoTypeSliceOfInterface(t *testing.T) {
	t.Parallel()
	inner := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	slice := pikoTypeOfSlice(inner)
	require.Equal(t, "[]main.myiface", slice.String())
	require.Equal(t, reflect.Slice, slice.Kind())
	require.Equal(t, "main.myiface", slice.Elem().String())
}

func TestPikoTypeMapOfInterface(t *testing.T) {
	t.Parallel()
	key := newPikoTypeFromReflect(reflect.TypeFor[string]())
	value := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	m := pikoTypeOfMap(key, value)
	require.Equal(t, "map[string]main.myiface", m.String())
	require.Equal(t, reflect.Map, m.Kind())
	require.Equal(t, "string", m.Key().String())
	require.Equal(t, "main.myiface", m.Elem().String())
}

func TestPikoTypeArrayOfInterface(t *testing.T) {
	t.Parallel()
	inner := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	arr := pikoTypeOfArray(5, inner)
	require.Equal(t, "[5]main.myiface", arr.String())
	require.Equal(t, reflect.Array, arr.Kind())
	require.Equal(t, 5, arr.Len())
}

func TestPikoTypeChanOfInterface(t *testing.T) {
	t.Parallel()
	inner := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	bidi := pikoTypeOfChan(reflect.BothDir, inner)
	require.Equal(t, "chan main.myiface", bidi.String())
	require.Equal(t, reflect.BothDir, bidi.ChanDir())

	recv := pikoTypeOfChan(reflect.RecvDir, inner)
	require.Equal(t, "<-chan main.myiface", recv.String())
	require.Equal(t, reflect.RecvDir, recv.ChanDir())

	send := pikoTypeOfChan(reflect.SendDir, inner)
	require.Equal(t, "chan<- main.myiface", send.String())
	require.Equal(t, reflect.SendDir, send.ChanDir())
}

func TestPikoTypeImplementsEmptyInterface(t *testing.T) {
	t.Parallel()
	user := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	empty := newPikoTypeNamedInterface("", "", nil)
	empty.kind = reflect.Interface
	require.True(t, user.Implements(empty), "every type implements the empty interface")
}

func TestPikoTypeImplementsSubsetMethodSet(t *testing.T) {
	t.Parallel()
	bigger := newPikoTypeNamedInterface("main", "Big", []string{"A", "B", "C"})
	smaller := newPikoTypeNamedInterface("main", "Small", []string{"A", "B"})
	require.True(t, bigger.Implements(smaller), "Big has A, B, C and Small wants A, B")
	require.False(t, smaller.Implements(bigger), "Small lacks C")
}

func TestPikoTypeAssignableToSameName(t *testing.T) {
	t.Parallel()
	a := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	b := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	require.True(t, a.AssignableTo(b))
	require.True(t, b.AssignableTo(a))
}

func TestPikoTypeFromReflectStdlibInterface(t *testing.T) {
	t.Parallel()
	pt := newPikoTypeFromReflect(reflect.TypeFor[error]())
	require.Equal(t, "error", pt.Name())
	require.Equal(t, "error", pt.String())
	require.Equal(t, reflect.Interface, pt.Kind())
	require.GreaterOrEqual(t, pt.NumMethod(), 1)
}

func TestPikoTypeFromReflectScalar(t *testing.T) {
	t.Parallel()
	pt := newPikoTypeFromReflect(reflect.TypeFor[int]())
	require.Equal(t, "int", pt.Name())
	require.Equal(t, reflect.Int, pt.Kind())
}

func TestPikoTypeFromReflectStruct(t *testing.T) {
	t.Parallel()
	type myStruct struct{ X int }
	pt := newPikoTypeFromReflect(reflect.TypeFor[myStruct]())
	require.Equal(t, "myStruct", pt.Name())
	require.Equal(t, reflect.Struct, pt.Kind())
	require.Equal(t, 1, pt.NumField())
	require.Equal(t, "X", pt.Field(0).Name)
}

func TestPikoTypeFuncSignature(t *testing.T) {
	t.Parallel()
	intT := newPikoTypeFromReflect(reflect.TypeFor[int]())
	stringT := newPikoTypeFromReflect(reflect.TypeFor[string]())
	errorT := newPikoTypeFromReflect(reflect.TypeFor[error]())

	fn := pikoTypeOfFunc([]*pikoType{intT, stringT}, []*pikoType{errorT}, false)
	require.Equal(t, reflect.Func, fn.Kind())
	require.Equal(t, 2, fn.NumIn())
	require.Equal(t, 1, fn.NumOut())
	require.Equal(t, "int", fn.In(0).Name())
	require.Equal(t, "string", fn.In(1).Name())
	require.Equal(t, "error", fn.Out(0).Name())
	require.False(t, fn.IsVariadic())
	require.Equal(t, "func(int, string) error", fn.String())
}

func TestPikoTypeFuncVariadic(t *testing.T) {
	t.Parallel()
	intT := newPikoTypeFromReflect(reflect.TypeFor[int]())
	intsT := pikoTypeOfSlice(intT)
	fn := pikoTypeOfFunc([]*pikoType{intT, intsT}, nil, true)
	require.Equal(t, "func(int, ...int)", fn.String())
	require.True(t, fn.IsVariadic())
}

func TestPikoTypeOverflow(t *testing.T) {
	t.Parallel()
	pt := newPikoTypeFromReflect(reflect.TypeFor[int8]())
	require.True(t, pt.OverflowInt(200))
	require.False(t, pt.OverflowInt(50))
}

func TestPikoTypeNilSafety(t *testing.T) {
	t.Parallel()
	var pt *pikoType
	require.Equal(t, "", pt.Name())
	require.Equal(t, "", pt.String())
	require.Equal(t, "", pt.PkgPath())
	require.Equal(t, reflect.Invalid, pt.Kind())
	require.Equal(t, 0, pt.NumMethod())
	require.Equal(t, 0, pt.NumField())
	require.Equal(t, 0, pt.NumIn())
	require.Equal(t, 0, pt.NumOut())
	require.Nil(t, pt.Elem())
	require.Nil(t, pt.Key())
	require.False(t, pt.Comparable())
}

func TestMethodSetSatisfies(t *testing.T) {
	t.Parallel()
	require.True(t, methodSetSatisfies([]string{"A", "B", "C"}, []string{"A", "C"}))
	require.True(t, methodSetSatisfies([]string{"A", "B"}, nil))
	require.False(t, methodSetSatisfies([]string{"A", "C"}, []string{"A", "B"}))
	require.False(t, methodSetSatisfies(nil, []string{"A"}))
}

func TestShortPackageName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "main", shortPackageName("main"))
	require.Equal(t, "bar", shortPackageName("github.com/foo/bar"))
	require.Equal(t, "", shortPackageName(""))
	require.Equal(t, "v1", shortPackageName("k8s.io/api/core/v1"))
}
