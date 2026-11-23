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

package annotator_domain

import (
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstituteType(t *testing.T) {
	t.Parallel()

	t.Run("returns original when expr is nil", func(t *testing.T) {
		t.Parallel()
		result := substituteType(nil, map[string]goast.Expr{"T": goast.NewIdent("int")})
		assert.Nil(t, result)
	})

	t.Run("returns original when substMap is empty", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		result := substituteType(expression, map[string]goast.Expr{})
		assert.Same(t, expression, result)
	})

	t.Run("returns original when substMap is nil", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		result := substituteType(expression, nil)
		assert.Same(t, expression, result)
	})

	t.Run("substitutes identifier T", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		replacement := goast.NewIdent("string")
		result := substituteType(expression, map[string]goast.Expr{"T": replacement})

		assert.Same(t, replacement, result)
	})

	t.Run("does not substitute non-matching identifier", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("NotT")
		result := substituteType(expression, map[string]goast.Expr{"T": goast.NewIdent("int")})

		assert.Same(t, expression, result)
	})
}

func TestSubstituteIdent(t *testing.T) {
	t.Parallel()

	t.Run("replaces matching identifier", func(t *testing.T) {
		t.Parallel()
		identifier := goast.NewIdent("K")
		replacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"K": replacement}

		result := substituteIdent(identifier, substMap)
		assert.Same(t, replacement, result)
	})

	t.Run("returns original for non-matching identifier", func(t *testing.T) {
		t.Parallel()
		identifier := goast.NewIdent("V")
		substMap := map[string]goast.Expr{"K": goast.NewIdent("string")}

		result := substituteIdent(identifier, substMap)
		assert.Same(t, identifier, result)
	})
}

func TestSubstituteStarExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes pointer element type", func(t *testing.T) {
		t.Parallel()
		starExpr := &goast.StarExpr{X: goast.NewIdent("T")}
		replacement := goast.NewIdent("User")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteStarExpr(starExpr, substMap)

		newStar, ok := result.(*goast.StarExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newStar.X)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		starExpr := &goast.StarExpr{X: goast.NewIdent("User")}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("int")}

		result := substituteStarExpr(starExpr, substMap)
		assert.Same(t, starExpr, result)
	})
}

func TestSubstituteArrayType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes slice element type", func(t *testing.T) {
		t.Parallel()
		arrayType := &goast.ArrayType{Elt: goast.NewIdent("T")}
		replacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteArrayType(arrayType, substMap)

		newArray, ok := result.(*goast.ArrayType)
		require.True(t, ok)
		assert.Same(t, replacement, newArray.Elt)
		assert.Nil(t, newArray.Len)
	})

	t.Run("preserves array length", func(t *testing.T) {
		t.Parallel()
		lenExpr := &goast.BasicLit{Kind: token.INT, Value: "10"}
		arrayType := &goast.ArrayType{Len: lenExpr, Elt: goast.NewIdent("T")}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteArrayType(arrayType, substMap)

		newArray, ok := result.(*goast.ArrayType)
		require.True(t, ok)
		assert.Same(t, lenExpr, newArray.Len)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		arrayType := &goast.ArrayType{Elt: goast.NewIdent("int")}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result := substituteArrayType(arrayType, substMap)
		assert.Same(t, arrayType, result)
	})
}

func TestSubstituteMapType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes both key and value types", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("K"),
			Value: goast.NewIdent("V"),
		}
		keyReplacement := goast.NewIdent("string")
		valueReplacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{
			"K": keyReplacement,
			"V": valueReplacement,
		}

		result := substituteMapType(mapType, substMap)

		newMap, ok := result.(*goast.MapType)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newMap.Key)
		assert.Same(t, valueReplacement, newMap.Value)
	})

	t.Run("substitutes only key type", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("K"),
			Value: goast.NewIdent("int"),
		}
		keyReplacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"K": keyReplacement}

		result := substituteMapType(mapType, substMap)

		newMap, ok := result.(*goast.MapType)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newMap.Key)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("int"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("float64")}

		result := substituteMapType(mapType, substMap)
		assert.Same(t, mapType, result)
	})
}

func TestSubstituteChanType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes channel element type", func(t *testing.T) {
		t.Parallel()
		chanType := &goast.ChanType{
			Dir:   goast.SEND | goast.RECV,
			Value: goast.NewIdent("T"),
		}
		replacement := goast.NewIdent("Message")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteChanType(chanType, substMap)

		newChan, ok := result.(*goast.ChanType)
		require.True(t, ok)
		assert.Same(t, replacement, newChan.Value)
		assert.Equal(t, goast.SEND|goast.RECV, newChan.Dir)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		chanType := &goast.ChanType{
			Dir:   goast.SEND,
			Value: goast.NewIdent("int"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result := substituteChanType(chanType, substMap)
		assert.Same(t, chanType, result)
	})
}

func TestSubstituteFieldList(t *testing.T) {
	t.Parallel()

	t.Run("returns nil and false for nil field list", func(t *testing.T) {
		t.Parallel()
		result, changed := substituteFieldList(nil, map[string]goast.Expr{"T": goast.NewIdent("int")})

		assert.Nil(t, result)
		assert.False(t, changed)
	})

	t.Run("substitutes field types", func(t *testing.T) {
		t.Parallel()
		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Names: []*goast.Ident{goast.NewIdent("x")}, Type: goast.NewIdent("T")},
			},
		}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result, changed := substituteFieldList(fieldList, substMap)

		require.True(t, changed)
		require.NotNil(t, result)
		assert.Same(t, replacement, result.List[0].Type)
	})

	t.Run("returns changed false when no substitution needed", func(t *testing.T) {
		t.Parallel()
		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Names: []*goast.Ident{goast.NewIdent("x")}, Type: goast.NewIdent("int")},
			},
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result, changed := substituteFieldList(fieldList, substMap)

		assert.False(t, changed)
		require.NotNil(t, result)
	})
}

func TestSubstituteIndexExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes generic type argument", func(t *testing.T) {
		t.Parallel()
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("Box"),
			Index: goast.NewIdent("T"),
		}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteIndexExpr(indexExpr, substMap)

		newIndex, ok := result.(*goast.IndexExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newIndex.Index)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("Box"),
			Index: goast.NewIdent("string"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("int")}

		result := substituteIndexExpr(indexExpr, substMap)
		assert.Same(t, indexExpr, result)
	})
}

func TestSubstituteIndexListExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes multiple type arguments", func(t *testing.T) {
		t.Parallel()
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("Map"),
			Indices: []goast.Expr{goast.NewIdent("K"), goast.NewIdent("V")},
		}
		keyReplacement := goast.NewIdent("string")
		valueReplacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{
			"K": keyReplacement,
			"V": valueReplacement,
		}

		result := substituteIndexListExpr(indexListExpr, substMap)

		newIndexList, ok := result.(*goast.IndexListExpr)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newIndexList.Indices[0])
		assert.Same(t, valueReplacement, newIndexList.Indices[1])
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("Map"),
			Indices: []goast.Expr{goast.NewIdent("string"), goast.NewIdent("int")},
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("float64")}

		result := substituteIndexListExpr(indexListExpr, substMap)
		assert.Same(t, indexListExpr, result)
	})
}

func TestSubstituteFuncType(t *testing.T) {
	t.Parallel()

	t.Run("returns original when no substitutions needed", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("int")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("float64"),
		}

		result := substituteFuncType(funcType, substMap)

		assert.Same(t, funcType, result, "should return original when no type params match")
	})

	t.Run("substitutes type parameter in params", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		require.NotNil(t, newFuncType.Params)
		require.Len(t, newFuncType.Params.List, 1)
		identifier, ok := newFuncType.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("substitutes type parameter in results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("U")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"U": goast.NewIdent("string"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		require.NotNil(t, newFuncType.Results)
		require.Len(t, newFuncType.Results.List, 1)
		identifier, ok := newFuncType.Results.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("substitutes in both params and results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("U")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
			"U": goast.NewIdent("string"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)

		paramIdent, ok := newFuncType.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", paramIdent.Name)

		resultIdent, ok := newFuncType.Results.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", resultIdent.Name)
	})

	t.Run("handles nil params", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: nil,
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		assert.Nil(t, newFuncType.Params)
	})

	t.Run("handles nil results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
			Results: nil,
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		assert.Nil(t, newFuncType.Results)
	})

	t.Run("handles empty substitution map", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{}

		result := substituteFuncType(funcType, substMap)

		assert.Same(t, funcType, result)
	})
}
