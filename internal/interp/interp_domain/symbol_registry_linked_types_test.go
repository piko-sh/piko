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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/interp/interp_link"
)

func TestMakeLinkedTypeParamsClampsCount(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test", "test")

	params := makeLinkedTypeParams(pkg, maxLinkedTypeArgCount+10)

	require.Len(t, params, maxLinkedTypeArgCount,
		"count beyond cap should be clamped to maxLinkedTypeArgCount")
}

func TestMakeLinkedTypeParamsZeroReturnsNil(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test", "test")
	require.Nil(t, makeLinkedTypeParams(pkg, 0))
	require.Nil(t, makeLinkedTypeParams(pkg, -1))
}

func TestMakeLinkedTypeParamsNamesUnique(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test", "test")

	params := makeLinkedTypeParams(pkg, 3)

	require.Len(t, params, 3)
	seen := make(map[string]struct{}, 3)
	for _, param := range params {
		name := param.Obj().Name()
		_, duplicate := seen[name]
		require.False(t, duplicate, "expected unique type-param names, got duplicate %q", name)
		seen[name] = struct{}{}
	}
}

func TestLinkedFieldToTypeReturnsNilAtDepthCap(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	descriptor := interp_link.GenericFieldType{
		Kind:      interp_link.FieldKindBasic,
		BasicKind: reflect.Int,
	}

	result := linkedFieldToType(registry, descriptor, nil, maxLinkedDescriptorDepth)
	require.Nil(t, result, "reaching the depth cap must return nil so the caller substitutes interface{}")
}

func TestLinkedFieldToTypeBasicKind(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)

	descriptor := interp_link.GenericFieldType{
		Kind:      interp_link.FieldKindBasic,
		BasicKind: reflect.String,
	}

	result := linkedFieldToType(registry, descriptor, nil, 0)
	require.NotNil(t, result)
	basic, ok := result.(*types.Basic)
	require.True(t, ok, "expected *types.Basic, got %T", result)
	require.Equal(t, types.String, basic.Kind())
}

func TestLinkedFieldToTypeSliceOfTypeArg(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)
	pkg := types.NewPackage("test", "test")
	typeParams := makeLinkedTypeParams(pkg, 1)

	descriptor := interp_link.GenericFieldType{
		Kind: interp_link.FieldKindSlice,
		Element: &interp_link.GenericFieldType{
			Kind:         interp_link.FieldKindTypeArg,
			TypeArgIndex: 0,
		},
	}

	result := linkedFieldToType(registry, descriptor, typeParams, 0)
	require.NotNil(t, result)
	slice, ok := result.(*types.Slice)
	require.True(t, ok, "expected *types.Slice, got %T", result)
	_, isTypeParam := slice.Elem().(*types.TypeParam)
	require.True(t, isTypeParam, "slice element should be the type parameter placeholder")
}

func TestLinkedFieldToTypeBadTypeArgIndexReturnsNil(t *testing.T) {
	t.Parallel()

	registry := NewSymbolRegistry(nil)
	pkg := types.NewPackage("test", "test")
	typeParams := makeLinkedTypeParams(pkg, 1)

	descriptor := interp_link.GenericFieldType{
		Kind:         interp_link.FieldKindTypeArg,
		TypeArgIndex: 5,
	}

	result := linkedFieldToType(registry, descriptor, typeParams, 0)
	require.Nil(t, result)
}

func TestRegisterNativeBackedGenericTypesMultiMethodNoPanic(t *testing.T) {
	registry := NewSymbolRegistry(nil)
	pkg := types.NewPackage("sync/atomic", "atomic")
	converter := &reflectTypeConverter{
		seen:       make(map[reflect.Type]types.Type),
		pkg:        pkg,
		registry:   registry,
		localTypes: make(map[reflect.Type]bool),
	}
	exports := map[string]reflect.Value{
		"Pointer": reflect.ValueOf(interp_link.WrapNativeGenericType(
			"Pointer", 1,
			reflect.TypeFor[atomic.Pointer[struct{}]](),
			reflect.TypeFor[struct{}]())),
	}

	require.NotPanics(t, func() {
		registry.registerNativeBackedGenericTypes(pkg, exports, converter)
	})

	object := pkg.Scope().Lookup("Pointer")
	require.NotNil(t, object)
	named, ok := object.Type().(*types.Named)
	require.True(t, ok)
	require.Equal(t, 1, named.TypeParams().Len())
	require.Equal(t, 4, named.NumMethods())
}
