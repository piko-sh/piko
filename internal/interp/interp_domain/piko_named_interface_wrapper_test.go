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

func TestPikoNamedInterfaceWrapperSatisfiesReflectType(t *testing.T) {
	t.Parallel()
	piko := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	wrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), piko)

	var asInterface reflect.Type = wrapper
	require.NotNil(t, asInterface)
}

func TestPikoNamedInterfaceWrapperReportsSourceName(t *testing.T) {
	t.Parallel()
	piko := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	wrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), piko)
	require.Equal(t, "myiface", wrapper.Name())
	require.Equal(t, "main.myiface", wrapper.String())
	require.Equal(t, "main", wrapper.PkgPath())
	require.Equal(t, reflect.Interface, wrapper.Kind())
}

func TestPikoNamedInterfaceWrapperPointerShape(t *testing.T) {
	t.Parallel()
	inner := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	ptr := pikoTypeOfPointer(inner)
	wrapper := newPikoNamedInterfaceWrapper(reflect.PointerTo(reflect.TypeFor[any]()), ptr)

	require.Equal(t, "*main.myiface", wrapper.String())
	require.Equal(t, "", wrapper.Name(), "pointer types have empty Name")
	require.Equal(t, reflect.Pointer, wrapper.Kind())

	elem := wrapper.Elem()
	require.Equal(t, "main.myiface", elem.String())
	require.Equal(t, reflect.Interface, elem.Kind())
	require.Equal(t, "myiface", elem.Name())
}

func TestPikoNamedInterfaceWrapperMethods(t *testing.T) {
	t.Parallel()
	piko := newPikoTypeNamedInterface("main", "myiface", []string{"Bar", "Foo"})
	wrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), piko)

	require.Equal(t, 2, wrapper.NumMethod())
	require.Equal(t, "Bar", wrapper.Method(0).Name)
	require.Equal(t, "Foo", wrapper.Method(1).Name)

	method, ok := wrapper.MethodByName("Foo")
	require.True(t, ok)
	require.Equal(t, "Foo", method.Name)

	_, ok = wrapper.MethodByName("Missing")
	require.False(t, ok)
}

func TestPikoNamedInterfaceWrapperImplements(t *testing.T) {
	t.Parallel()
	bigger := newPikoTypeNamedInterface("main", "Big", []string{"A", "B", "C"})
	bigWrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), bigger)

	smaller := newPikoTypeNamedInterface("main", "Small", []string{"A", "B"})
	smallWrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), smaller)

	require.True(t, bigWrapper.Implements(smallWrapper))
	require.False(t, smallWrapper.Implements(bigWrapper))

	require.True(t, bigWrapper.Implements(reflect.TypeFor[any]()))
}

func TestPikoNamedInterfaceWrapperUnwrap(t *testing.T) {
	t.Parallel()
	piko := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	native := reflect.TypeFor[any]()
	wrapper := newPikoNamedInterfaceWrapper(native, piko)

	unwrapped, ok := unwrapPikoNamedInterfaceWrapper(wrapper)
	require.True(t, ok)
	require.Equal(t, native, unwrapped.Type)

	_, ok = unwrapPikoNamedInterfaceWrapper(reflect.TypeFor[int]())
	require.False(t, ok)

	require.Equal(t, native, nativeReflectTypeFromWrapper(wrapper))
	require.Equal(t, reflect.TypeFor[int](), nativeReflectTypeFromWrapper(reflect.TypeFor[int]()))
}

func TestPikoNamedInterfaceWrapperEmbeddedDelegation(t *testing.T) {
	t.Parallel()

	piko := newPikoTypeNamedInterface("main", "myiface", []string{"Foo"})
	wrapper := newPikoNamedInterfaceWrapper(reflect.TypeFor[any](), piko)

	require.True(t, wrapper.Comparable(), "interfaces are comparable")
}
