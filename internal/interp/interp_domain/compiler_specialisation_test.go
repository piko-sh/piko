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
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeTypeParam(name string) *types.TypeParam {
	tname := types.NewTypeName(token.NoPos, nil, name, nil)
	anyType := types.Universe.Lookup("any").Type()
	constraint, ok := anyType.Underlying().(*types.Interface)
	if !ok {
		panic("any's underlying type is not an interface -- go/types invariant broken")
	}
	return types.NewTypeParam(tname, constraint)
}

func TestSubstituteTypeNilReturnsNil(t *testing.T) {
	t.Parallel()
	require.Nil(t, substituteType(nil, map[*types.TypeParam]types.Type{}, nil))
}

func TestSubstituteTypeEmptySubsReturnsIdentity(t *testing.T) {
	t.Parallel()
	intType := types.Typ[types.Int]
	require.Same(t, intType, substituteType(intType, nil, nil))
	require.Same(t, intType, substituteType(intType, map[*types.TypeParam]types.Type{}, nil))
}

func TestSubstituteTypeBareTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	subs := map[*types.TypeParam]types.Type{tp: intType}
	require.Same(t, intType, substituteType(tp, subs, nil))
}

func TestSubstituteTypeUnmappedTypeParamPassesThrough(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	other := makeTypeParam("K")
	subs := map[*types.TypeParam]types.Type{other: types.Typ[types.String]}
	require.Same(t, tp, substituteType(tp, subs, nil))
}

func TestSubstituteTypePointerOfTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	pointer := types.NewPointer(tp)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(pointer, subs, map[types.Type]types.Type{})
	resultPointer, ok := result.(*types.Pointer)
	require.True(t, ok, "result must be a *types.Pointer")
	require.Same(t, intType, resultPointer.Elem(),
		"the pointer's element type must be the substituted concrete type -- without this every generic path that takes a pointer parameter would fail to specialise correctly")
}

func TestSubstituteTypeSliceOfTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	slice := types.NewSlice(tp)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(slice, subs, map[types.Type]types.Type{})
	resultSlice, ok := result.(*types.Slice)
	require.True(t, ok, "result must be a *types.Slice")
	require.Same(t, intType, resultSlice.Elem(),
		"slice element substitution must recurse -- the body of a generic that ranges []T relies on this for typed-bank slice ops")
}

func TestSubstituteTypeMapOfTypeParams(t *testing.T) {
	t.Parallel()
	keyParam := makeTypeParam("K")
	valueParam := makeTypeParam("V")
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]
	mapType := types.NewMap(keyParam, valueParam)
	subs := map[*types.TypeParam]types.Type{keyParam: stringType, valueParam: intType}
	result := substituteType(mapType, subs, map[types.Type]types.Type{})
	resultMap, ok := result.(*types.Map)
	require.True(t, ok, "result must be a *types.Map")
	require.Same(t, stringType, resultMap.Key(),
		"map key substitution -- both K and V must resolve independently to support multi-param generics")
	require.Same(t, intType, resultMap.Elem(),
		"map value substitution")
}

func TestSubstituteTypeArrayOfTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	array := types.NewArray(tp, 5)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(array, subs, map[types.Type]types.Type{})
	resultArray, ok := result.(*types.Array)
	require.True(t, ok, "result must be a *types.Array")
	require.Same(t, intType, resultArray.Elem(),
		"array element substitution preserves the length while replacing the element type")
	require.Equal(t, int64(5), resultArray.Len(),
		"array length must round-trip -- a generic [N]T with concrete N stays the same N after substitution")
}

func TestSubstituteTypeChanOfTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	chanType := types.NewChan(types.SendOnly, tp)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(chanType, subs, map[types.Type]types.Type{})
	resultChan, ok := result.(*types.Chan)
	require.True(t, ok, "result must be a *types.Chan")
	require.Same(t, intType, resultChan.Elem(),
		"channel element substitution -- generic chan T must specialise to concrete chan int")
	require.Equal(t, types.SendOnly, resultChan.Dir(),
		"channel direction must be preserved")
}

func TestSubstituteTypeNestedComposite(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	pointerToTP := types.NewPointer(tp)
	sliceOfPointer := types.NewSlice(pointerToTP)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(sliceOfPointer, subs, map[types.Type]types.Type{})
	resultSlice, ok := result.(*types.Slice)
	require.True(t, ok)
	resultPointer, ok := resultSlice.Elem().(*types.Pointer)
	require.True(t, ok)
	require.Same(t, intType, resultPointer.Elem(),
		"deeply nested substitution must propagate through every layer -- without this []*Box[T] in a generic body would fail to specialise correctly")
}

func TestSubstituteTypeIdentityWhenNoSubstitutionApplies(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	stringType := types.Typ[types.String]
	otherTP := makeTypeParam("U")
	pointerToOther := types.NewPointer(otherTP)
	subs := map[*types.TypeParam]types.Type{tp: stringType}
	require.Same(t, pointerToOther, substituteType(pointerToOther, subs, map[types.Type]types.Type{}),
		"unchanged sub-trees must be returned via identity to avoid unnecessary allocation churn during specialisation")
}

func TestSubstituteTypeStructFieldWithTypeParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	field := types.NewField(token.NoPos, nil, "Value", tp, false)
	structType := types.NewStruct([]*types.Var{field}, []string{`json:"value"`})
	subs := map[*types.TypeParam]types.Type{tp: intType}
	result := substituteType(structType, subs, map[types.Type]types.Type{})
	resultStruct, ok := result.(*types.Struct)
	require.True(t, ok)
	require.Equal(t, 1, resultStruct.NumFields())
	require.Same(t, intType, resultStruct.Field(0).Type(),
		"struct field substitution -- Box[T]{Value T} must specialise to Box[int]{Value int}")
	require.Equal(t, `json:"value"`, resultStruct.Tag(0),
		"struct field tags must be preserved across substitution")
}

func TestSubstituteTypeSignatureWithTypeParamParam(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	param := types.NewVar(token.NoPos, nil, "x", tp)
	result := types.NewVar(token.NoPos, nil, "", tp)
	signature := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(param), types.NewTuple(result), false)
	subs := map[*types.TypeParam]types.Type{tp: intType}
	substituted := substituteType(signature, subs, map[types.Type]types.Type{})
	resultSig, ok := substituted.(*types.Signature)
	require.True(t, ok)
	require.Equal(t, 1, resultSig.Params().Len())
	require.Same(t, intType, resultSig.Params().At(0).Type(),
		"signature param substitution -- func(T) T must specialise to func(int) int when T->int")
	require.Equal(t, 1, resultSig.Results().Len())
	require.Same(t, intType, resultSig.Results().At(0).Type(),
		"signature result substitution")
}

func TestSubstituteTypeCacheReusesResults(t *testing.T) {
	t.Parallel()
	tp := makeTypeParam("T")
	intType := types.Typ[types.Int]
	slice := types.NewSlice(tp)
	cache := map[types.Type]types.Type{}
	subs := map[*types.TypeParam]types.Type{tp: intType}
	first := substituteType(slice, subs, cache)
	second := substituteType(slice, subs, cache)
	require.Same(t, first, second,
		"the cache must return the same constructed type on repeated lookups -- without this, deeply nested generic bodies would allocate a fresh substituted slice/map/etc. for every reference, multiplying the substitution cost")
}
