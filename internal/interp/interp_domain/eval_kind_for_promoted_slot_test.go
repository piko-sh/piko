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
	"context"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKindForPromotedSlotPredicate(t *testing.T) {
	t.Parallel()

	sliceInt64 := types.NewSlice(types.Typ[types.Int64])
	scalarInt := types.Typ[types.Int64]

	t.Run("nil_ctx_falls_through_to_kindForCallSlot", func(t *testing.T) {
		t.Parallel()
		got, promoted := kindForPromotedSlot(sliceInt64, nil)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("nil_ctx_non_slice_type_falls_through", func(t *testing.T) {
		t.Parallel()
		got, promoted := kindForPromotedSlot(scalarInt, nil)
		require.Equal(t, registerInt, got)
		require.False(t, promoted, "scalar types never promote to typed-slice banks")
	})

	t.Run("empty_promoted_vector_falls_through", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: nil}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("paramIdx_out_of_range_falls_through", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: []bool{true}, calleeParamIndex: 5}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted, "out-of-range index falls through to type-only verdict")
	})

	t.Run("negative_paramIdx_falls_through", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: []bool{true}, calleeParamIndex: -1}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("promoted_true_keeps_typed_bank", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: []bool{true}, calleeParamIndex: 0}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("promoted_false_demotes_to_general", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: []bool{false}, calleeParamIndex: 0}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerGeneral, got)
		require.False(t, promoted)
	})

	t.Run("non_slice_type_unaffected_by_demotion", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{calleeParamPromotions: []bool{false}, calleeParamIndex: 0}
		got, promoted := kindForPromotedSlot(scalarInt, ctx)
		require.Equal(t, registerInt, got)
		require.False(t, promoted)
	})

	t.Run("disqualifier_set_demotes_named_binding", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{
			disqualified: map[string]bool{"xs": true},
			bindingName:  "xs",
		}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerGeneral, got)
		require.False(t, promoted)
	})

	t.Run("disqualifier_set_ignores_unrelated_binding", func(t *testing.T) {
		t.Parallel()
		ctx := &kindPromotionContext{
			disqualified: map[string]bool{"otherVar": true},
			bindingName:  "xs",
		}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("empty_bindingName_skips_disqualifier_gate", func(t *testing.T) {
		t.Parallel()

		ctx := &kindPromotionContext{
			disqualified: map[string]bool{"anyName": true},
			bindingName:  "",
		}
		got, promoted := kindForPromotedSlot(sliceInt64, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("substitution_resolves_typeparam_to_concrete", func(t *testing.T) {
		t.Parallel()

		typeParam := types.NewTypeParam(types.NewTypeName(0, nil, "V", nil), nil)
		sliceOfV := types.NewSlice(typeParam)
		ctx := &kindPromotionContext{
			substitutions: map[*types.TypeParam]types.Type{
				typeParam: types.Typ[types.Int64],
			},
		}
		got, promoted := kindForPromotedSlot(sliceOfV, ctx)
		require.Equal(t, registerSliceInt, got)
		require.True(t, promoted)
	})

	t.Run("substitution_path_respects_disqualifier", func(t *testing.T) {
		t.Parallel()
		typeParam := types.NewTypeParam(types.NewTypeName(0, nil, "V", nil), nil)
		sliceOfV := types.NewSlice(typeParam)
		ctx := &kindPromotionContext{
			substitutions: map[*types.TypeParam]types.Type{
				typeParam: types.Typ[types.Int64],
			},
			disqualified: map[string]bool{"xs": true},
			bindingName:  "xs",
		}
		got, promoted := kindForPromotedSlot(sliceOfV, ctx)
		require.Equal(t, registerGeneral, got)
		require.False(t, promoted)
	})
}

func TestParameterTypedSlicePromotedPublishedNonSpecialised(t *testing.T) {
	t.Parallel()

	source := `package main

// xs survives typed-bank because the body only does index reads.
func surviveByIndex(xs []int64) int64 {
	var total int64
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}

// xs is demoted by the spread-append disqualifier (typed-direct
// append only accepts single-element shape).
func demoteBySpread(xs []int64, more []int64) []int64 {
	return append(xs, more...)
}

// xs is demoted by the address-of disqualifier.
func demoteByAddressof(xs []int64) []int64 {
	p := &xs
	return *p
}

func main() {
	xs := []int64{1, 2, 3}
	_ = surviveByIndex(xs)
	_ = demoteBySpread(xs, xs)
	_ = demoteByAddressof(xs)
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	survive := findCompiledFunctionByNameForPromotedTest(t, cfs, "surviveByIndex")
	require.NotNil(t, survive)
	require.Len(t, survive.parameterTypedSlicePromoted, 1)
	require.True(t, survive.parameterTypedSlicePromoted[0],
		"surviveByIndex.xs should be parameterTypedSlicePromoted[0]=true (stayed on registerSliceInt)")
	require.Equal(t, registerSliceInt, survive.parameterKinds[0],
		"sanity: parameterKinds[0] should be registerSliceInt for the survived parameter")

	spread := findCompiledFunctionByNameForPromotedTest(t, cfs, "demoteBySpread")
	require.NotNil(t, spread)
	require.GreaterOrEqual(t, len(spread.parameterTypedSlicePromoted), 1)
	require.False(t, spread.parameterTypedSlicePromoted[0],
		"demoteBySpread.xs should be parameterTypedSlicePromoted[0]=false (demoted by spread-append)")
	require.Equal(t, registerGeneral, spread.parameterKinds[0],
		"sanity: parameterKinds[0] should be registerGeneral for the demoted parameter")

	addr := findCompiledFunctionByNameForPromotedTest(t, cfs, "demoteByAddressof")
	require.NotNil(t, addr)
	require.Len(t, addr.parameterTypedSlicePromoted, 1)
	require.False(t, addr.parameterTypedSlicePromoted[0],
		"demoteByAddressof.xs should be parameterTypedSlicePromoted[0]=false (demoted by &xs)")
}

func TestParameterTypedSlicePromotedPublishedSpecialised(t *testing.T) {
	t.Parallel()

	source := `package main

func gsum[T ~int64](xs []T) T {
	var total T
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}

func gAppendSpread[T ~int64](xs []T, more []T) []T {
	return append(xs, more...)
}

func main() {
	xs := []int64{1, 2, 3}
	_ = gsum(xs)
	_ = gAppendSpread(xs, xs)
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	gsumSpec := findCompiledFunctionBySpecPrefixForPromotedTest(t, cfs, "gsum")
	require.NotNil(t, gsumSpec, "expected a specialisation of gsum")
	require.GreaterOrEqual(t, len(gsumSpec.parameterTypedSlicePromoted), 1)
	require.True(t, gsumSpec.parameterTypedSlicePromoted[0],
		"specialised gsum.xs should stay typed-bank")
	require.Equal(t, registerSliceInt, gsumSpec.parameterKinds[0])

	spreadSpec := findCompiledFunctionBySpecPrefixForPromotedTest(t, cfs, "gAppendSpread")
	require.NotNil(t, spreadSpec, "expected a specialisation of gAppendSpread")
	require.GreaterOrEqual(t, len(spreadSpec.parameterTypedSlicePromoted), 1)
	require.False(t, spreadSpec.parameterTypedSlicePromoted[0],
		"specialised gAppendSpread.xs should be demoted (spread-append disqualifier)")
	require.Equal(t, registerGeneral, spreadSpec.parameterKinds[0])
}

func findCompiledFunctionByNameForPromotedTest(t *testing.T, cfs *CompiledFileSet, name string) *CompiledFunction {
	t.Helper()
	fn, err := cfs.FindFunction(name)
	if err != nil {
		return nil
	}
	return fn
}

func findCompiledFunctionBySpecPrefixForPromotedTest(t *testing.T, cfs *CompiledFileSet, origin string) *CompiledFunction {
	t.Helper()
	prefix := origin + "[spec:"
	for _, fn := range cfs.root.functions {
		if fn == nil {
			continue
		}
		if strings.HasPrefix(fn.name, prefix) {
			return fn
		}
	}
	return nil
}
