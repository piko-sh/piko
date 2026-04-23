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

package interp_link_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/interp/interp_link"
)

func sampleSibling(typeArg reflect.Type, multiplier int) reflect.Value {
	return reflect.New(typeArg).Elem()
}

func TestWrapStoresTargetAndCount(t *testing.T) {
	t.Parallel()

	linked := interp_link.Wrap(1, sampleSibling)

	require.Equal(t, 1, linked.TypeArgCount)
	require.True(t, linked.Target.IsValid())
	require.Equal(t, reflect.Func, linked.Target.Kind())
}

func TestWrapInvokesThroughReflectValue(t *testing.T) {
	t.Parallel()

	linked := interp_link.Wrap(1, sampleSibling)

	stringType := reflect.TypeFor[string]()
	results := linked.Target.Call([]reflect.Value{
		reflect.ValueOf(stringType),
		reflect.ValueOf(42),
	})

	require.Len(t, results, 1)

	inner, ok := results[0].Interface().(reflect.Value)
	require.True(t, ok, "sibling must return a reflect.Value")
	require.Equal(t, stringType, inner.Type())
}

func TestWrapWithZeroTypeArgs(t *testing.T) {
	t.Parallel()

	plain := func(multiplier int) int { return multiplier * 2 }
	linked := interp_link.Wrap(0, plain)

	require.Equal(t, 0, linked.TypeArgCount)
	results := linked.Target.Call([]reflect.Value{reflect.ValueOf(5)})
	require.Len(t, results, 1)
	require.Equal(t, 10, int(results[0].Int()))
}

func TestWrapFuncCapturesShape(t *testing.T) {
	t.Parallel()

	sibling := func(typeArg reflect.Type, values []int) (reflect.Value, error) {
		return reflect.ValueOf(len(values)), nil
	}
	params := []interp_link.GenericFieldType{
		{
			Kind: interp_link.FieldKindSlice,
			Element: &interp_link.GenericFieldType{
				Kind:      interp_link.FieldKindBasic,
				BasicKind: reflect.Int,
			},
		},
	}
	results := []interp_link.GenericFieldType{
		{Kind: interp_link.FieldKindTypeArg, TypeArgIndex: 0},
		{Kind: interp_link.FieldKindError},
	}

	linked := interp_link.WrapFunc(1, sibling, params, results, true)

	require.Equal(t, 1, linked.TypeArgCount)
	require.True(t, linked.Variadic)
	require.Equal(t, params, linked.Params)
	require.Equal(t, results, linked.Results)
	require.Equal(t, reflect.Func, linked.Target.Kind())
}

func TestWrapTypeCapturesGenericFields(t *testing.T) {
	t.Parallel()

	fields := []interp_link.GenericField{
		{
			Name:      "Value",
			FieldType: interp_link.GenericFieldType{Kind: interp_link.FieldKindTypeArg, TypeArgIndex: 0},
			Exported:  true,
		},
		{
			Name:      "Score",
			FieldType: interp_link.GenericFieldType{Kind: interp_link.FieldKindBasic, BasicKind: reflect.Float64},
			Exported:  true,
		},
	}

	linkedType := interp_link.WrapType("Box", 1, fields)

	require.Equal(t, "Box", linkedType.Name)
	require.Equal(t, 1, linkedType.TypeArgCount)
	require.Equal(t, fields, linkedType.Fields)
}
